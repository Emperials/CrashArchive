package handler

import (
	"log"
	"net/http"
	"net/url"

	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/pmmp/CrashArchive/app/crashreport"
	"github.com/pmmp/CrashArchive/app/database"
	"github.com/pmmp/CrashArchive/app/template"
)

func ListGet(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			filter, filterParams, err := buildSearchQuery(r.URL.Query())
			if err != nil {
				template.ErrorTemplate(w, r, "", http.StatusBadRequest)
				return
			}
			log.Printf("search generated query: %s\n", filter)
			ListFilteredReports(w, r, db, filter, filterParams...)
		}
	}
}

func parseUintParam(v url.Values, paramName string, defaultValue uint64) (uint64, error) {
	param := v.Get(paramName)
	if param != "" {
		retval, err := strconv.ParseUint(param, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid value for search parameter \"%s\"", paramName)
		}

		return retval, nil
	}

	return defaultValue, nil
}

func buildSearchQuery(params url.Values) (string, []interface{}, error) {
	filters := make([]string, 0)
	filterParams := make([]interface{}, 0)
	var filter string

	if params.Get("duplicates") != "true" {
		filters = append(filters, "duplicate = false")
	}

	if params.Get("min") != "" || params.Get("max") != "" {
		//check ranges
		filterMinId, err := parseUintParam(params, "min", 0)
		if err != nil {
			return "", nil, err
		}
		filterMaxId, err := parseUintParam(params, "max", math.MaxUint64)
		if err != nil {
			return "", nil, err
		}

		if filterMinId > filterMaxId {
			return "", nil, errors.New("Invalid min/max ID bounds")
		}

		filters = append(filters, "id BETWEEN ? AND ?")
		filterParams = append(filterParams, filterMinId, filterMaxId)
	}

	//filter by message
	message := params.Get("message")
	if message != "" {
		filters = append(filters, "message LIKE ?")
		filterParams = append(filterParams, "%"+message+"%")
	}

	errortype := params.Get("errortype")
	if errortype != "" {
		filters = append(filters, "type LIKE ?")
		filterParams = append(filterParams, "%"+errortype+"%")
	}

	if causes := params["cause"]; causes != nil && len(causes) > 0 {
		involvements := []string{}
		for _, cause := range causes {
			var involvement string = ""
			switch cause {
			case "core":
				involvement = crashreport.PINone
			case "plugin":
				involvement = crashreport.PIDirect
			case "plugin_indirect":
				involvement = crashreport.PIIndirect
			default:
				return "", nil, fmt.Errorf("Invalid cause filter %s", cause)
			}
			involvements = append(involvements, involvement)
		}
		qs := strings.Repeat("?,", len(involvements))
		filters = append(filters, fmt.Sprintf("pluginInvolvement IN (%s)", qs[:len(qs)-1]))
		for _, involvement := range involvements {
			filterParams = append(filterParams, involvement)
		}
	}

	//filter by plugin
	plugin := params.Get("plugin")
	if plugin != "" {
		filters = append(filters, "plugin = ?")
		filterParams = append(filterParams, plugin)
	}

	//filter by build number
	if params.Get("build") != "" {
		buildID, err := parseUintParam(params, "build", math.MaxUint64)
		if err != nil {
			return "", nil, err
		}

		operator := "="
		typ := params.Get("buildtype")
		if typ == "greater" {
			operator = ">"
		} else if typ == "less" {
			operator = "<"
		}

		filters = append(filters, fmt.Sprintf("build %s ?", operator))
		filterParams = append(filterParams, buildID)
	}

	if filterVersions := params["versions"]; filterVersions != nil && len(filterVersions) > 0 {
		qs := strings.Repeat("?,", len(filterVersions))
		filters = append(filters, fmt.Sprintf("version IN (%s)", qs[:len(qs)-1]))
		for _, filterVersion := range filterVersions {
			filterParams = append(filterParams, filterVersion)
		}
	}

	filter = strings.Join(filters[:], " AND ")
	if filter != "" {
		filter = "WHERE " + filter
	}
	return filter, filterParams, nil
}

func ListFilteredReports(w http.ResponseWriter, r *http.Request, db *database.DB, filter string, filterParams ...interface{}) {
	var err error
	var total int

	queryCount := fmt.Sprintf("SELECT COUNT(*) FROM crash_reports %s", filter)
	err = db.Get(&total, queryCount, filterParams...)
	if err != nil {
		log.Println(err)
		log.Println(queryCount)
		template.ErrorTemplate(w, r, "", http.StatusInternalServerError)
		return
	}

	pageId := 1
	pageSize := 50

	params := r.URL.Query()
	if pageSizeParam := params.Get("pagesize"); pageSizeParam != "" {
		pageSize, err = strconv.Atoi(pageSizeParam)
		if err != nil || pageSize <= 0 || pageSize > 1000 {
			template.ErrorTemplate(w, r, "Illegal page size parameter", http.StatusBadRequest)
			return
		}
	}

	if pageParam := params.Get("page"); pageParam != "" {
		pageId, err = strconv.Atoi(pageParam)
		if err != nil || pageId <= 0 || (pageId-1)*pageSize > total {
			template.ErrorTemplate(w, r, "", http.StatusNotFound)
			return
		}
	}

	rangeStart := (pageId - 1) * pageSize

	var reports []crashreport.Report
	querySelect := fmt.Sprintf("SELECT id, version, plugin, message, resolved FROM crash_reports %s ORDER BY id DESC LIMIT %d, %d", filter, rangeStart, pageSize)
	err = db.Select(&reports, querySelect, filterParams...)
	if err != nil {
		log.Println(err)
		log.Println(querySelect)
		template.ErrorTemplate(w, r, "", http.StatusInternalServerError)
		return
	}

	template.ExecuteListTemplate(w, r, reports, r.URL.String(), pageId, rangeStart, total)
}

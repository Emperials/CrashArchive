package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/go-chi/chi"

	"github.com/pmmp/CrashArchive/app"
	"github.com/pmmp/CrashArchive/app/database"
	"github.com/pmmp/CrashArchive/app/template"
	"github.com/pmmp/CrashArchive/app/user"
)

func ViewIDGet(db *database.DB, config *app.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			reportID, err := strconv.Atoi(chi.URLParam(r, "reportID"))
			if err != nil {
				template.ErrorTemplate(w, r, "Please specify a report", http.StatusNotFound)
				return
			}

			var reporterName string
			err = db.Get(&reporterName, "SELECT reporterName FROM crash_reports WHERE id = ?", reportID)
			if err != nil {
				log.Printf("can't find report %d in database: %v", reportID, err)
				template.ErrorTemplate(w, r, "Report not found", http.StatusNotFound)
				return
			}

			report, err := db.FetchReport(int64(reportID))
			if err != nil {
				log.Printf("error fetching report: %v", err)
				template.ErrorTemplate(w, r, "Report not found", http.StatusNotFound)
				return
			}
			resolved, err := db.CheckResolved(int64(reportID))
			if err != nil {
				log.Printf("error checking if resolved: %v", err)
				template.ErrorTemplate(w, r, "Report not found", http.StatusNotFound)
				return
			}

			v := make(map[string]interface{})
			v["Report"] = report
			v["Name"] = clean(reporterName)
			v["PocketMineVersion"] = report.Version.Get(true)
			v["ReportID"] = reportID
			v["HasDeletePerm"] = user.GetUserInfo(r).HasDeletePerm()
			v["Resolved"] = resolved

			issueQueryParams := url.Values{}
			issueQueryParams.Add("title", report.Error.Message)
			issueQueryParams.Add("body", fmt.Sprintf("Link to crashdump: %s/view/%d\n\n### Additional comments\n", config.Domain, reportID))

			template.ExecuteTemplateParams(w, r, "view", v)
		}
	}
}

func ViewIDRawGet(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			reportID, err := strconv.Atoi(chi.URLParam(r, "reportID"))
			if err != nil {
				template.ErrorTemplate(w, r, "Please specify a report", http.StatusNotFound)
				return
			}

			report, err := db.FetchRawReport(int64(reportID))
			if err != nil {
				log.Printf("error fetching report: %v", err)
				template.ErrorTemplate(w, r, "Report not found", http.StatusNotFound)
				return
			}

			var buffer bytes.Buffer
			json.Indent(&buffer, report, "", "    ")
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write(buffer.Bytes())
		}
	}
}

var cleanRE = regexp.MustCompile(`[^A-Za-z0-9_\-\.\,\;\:/\#\(\)\\ +]`)

func clean(v string) string {
	return cleanRE.ReplaceAllString(v, "")
}

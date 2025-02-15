package handler

import (
	"facette.io/natsort"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/pmmp/CrashArchive/app/crashreport"
	"github.com/pmmp/CrashArchive/app/database"
	"github.com/pmmp/CrashArchive/app/template"
)

func SearchGet(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			knownVersions := []string{}
			err := db.Select(&knownVersions, `SELECT DISTINCT version FROM crash_reports`)
			if err != nil {
				fmt.Printf("error fetching known versions: %v\n", err)
				template.ErrorTemplate(w, r, "", http.StatusInternalServerError)
				return
			}
			log.Printf("Found %d known versions\n", len(knownVersions))
			natsort.Sort(knownVersions)
			args := make(map[string]interface{})
			args["KnownVersions"] = knownVersions
			template.ExecuteTemplateParams(w, r, "search", args)
		}
	}
}

func SearchIDGet(w http.ResponseWriter, r *http.Request) {
	if requireLogin(w, r) {
		reportID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			log.Println(err)
			template.ErrorTemplate(w, r, "", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/view/%d", reportID), http.StatusMovedPermanently)
	}
}

func SearchReportGet(db *database.DB) http.HandlerFunc {
	query := "SELECT * FROM crash_reports WHERE id = ?"
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			reportID, err := strconv.Atoi(r.URL.Query().Get("id"))
			if err != nil {
				log.Println(err)
				template.ErrorTemplate(w, r, "", http.StatusBadRequest)
				return
			}

			var report crashreport.Report
			err = db.Get(&report, query, reportID)
			if err != nil {
				log.Println(err)
				template.ErrorTemplate(w, r, "Report not found", http.StatusNotFound)
				return
			}

			ListFilteredReports(w, r, db, "WHERE message = ? AND file = ? and line = ?", report.Message, report.File, report.Line)
		}
	}
}

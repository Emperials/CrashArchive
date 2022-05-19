package handler

import (
	"github.com/go-chi/chi"
	"github.com/pmmp/CrashArchive/app/database"
	"github.com/pmmp/CrashArchive/app/template"
	"github.com/pmmp/CrashArchive/app/user"
	"log"
	"net/http"
	"strconv"
)

func ResolveGet(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireLogin(w, r) {
			userInfo := user.GetUserInfo(r)
			reportID, err := strconv.Atoi(chi.URLParam(r, "reportID"))
			if err != nil {
				template.ErrorTemplate(w, r, "Please specify a report", http.StatusNotFound)
				return
			}

			db.Exec("UPDATE crash_reports SET resolved=1 WHERE id = ? OR duplicatedId = ?", reportID, reportID)
			log.Printf("user %s resolved crash report %d", userInfo.Name, reportID)
			redirectUrl := r.URL.Query().Get("redirect")
			if redirectUrl == "" {
				redirectUrl = "/list"
			}
			http.Redirect(w, r, redirectUrl, http.StatusMovedPermanently)
		}
	}
}

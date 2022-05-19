package handler

import (
	"net/http"

	"github.com/pmmp/CrashArchive/app/template"
)

func HomeGet(w http.ResponseWriter, r *http.Request) {
	if requireLogin(w, r) {
		template.ExecuteTemplate(w, r, "home")
	}
}

package api

import "net/http"

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) error {
	return sendJSON(w, http.StatusOK, a.HealthCheck())
}

func (a *API) HealthCheck() map[string]string {
	return map[string]string{
		"version":     a.version,
		"name":        "GoTrue",
		"description": "GoTrue is a user registration and authentication API",
	}
}

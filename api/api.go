package api

import (
	"net/http"

	"github.com/guregu/kami"
	"github.com/netlify/authlify/db"
	"github.com/rs/cors"
)

// API is the main REST API
type API struct {
	handler http.Handler
	db      *db.DB
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a.handler)
}

// NewAPI instantiates a new REST API
func NewAPI(db *db.DB) *API {
	api := &API{db: db}
	mux := kami.New()

	mux.Get("/", api.Index)
	mux.Post("/signup", api.Signup)
	mux.Post("/verify", api.Verify)
	mux.Post("/login", api.Login)
	mux.Post("/logout", api.Logout)

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(mux)
	return api
}

package api

import (
	"net/http"

	"github.com/guregu/kami"
	"github.com/jinzhu/gorm"
	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/mailer"
	"github.com/rs/cors"
)

// API is the main REST API
type API struct {
	handler http.Handler
	db      *gorm.DB
	mailer  *mailer.Mailer
	config  *conf.Configuration
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a.handler)
}

// NewAPI instantiates a new REST API
func NewAPI(config *conf.Configuration, db *gorm.DB, mailer *mailer.Mailer) *API {
	api := &API{config: config, db: db, mailer: mailer}
	mux := kami.New()

	mux.Get("/", api.Index)
	mux.Post("/signup", api.Signup)
	mux.Post("/verify", api.Verify)
	mux.Post("/token", api.Token)
	mux.Post("/logout", api.Logout)

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(mux)
	return api
}

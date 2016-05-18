package api

import (
	"fmt"
	"net/http"
	"regexp"

	"golang.org/x/net/context"

	"github.com/dgrijalva/jwt-go"
	"github.com/guregu/kami"
	"github.com/jinzhu/gorm"
	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/mailer"
	"github.com/rs/cors"
)

var bearerRegexp = regexp.MustCompile(`^(?:B|b)earer (\S+$)`)

// API is the main REST API
type API struct {
	handler http.Handler
	db      *gorm.DB
	mailer  *mailer.Mailer
	config  *conf.Configuration
}

func (a *API) requireAuthentication(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return nil
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return nil
	}

	token, err := jwt.Parse(matches[1], func(token *jwt.Token) (interface{}, error) {
		if token.Header["alg"] != "HS256" {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.JWT.Secret), nil
	})
	if err != nil {
		UnauthorizedError(w, fmt.Sprintf("Invalid token: %v", err))
		return nil
	}

	return context.WithValue(ctx, "jwt", token)
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a.handler)
}

// NewAPI instantiates a new REST API
func NewAPI(config *conf.Configuration, db *gorm.DB, mailer *mailer.Mailer) *API {
	api := &API{config: config, db: db, mailer: mailer}
	mux := kami.New()

	mux.Use("/user", api.requireAuthentication)
	mux.Use("/logout", api.requireAuthentication)

	mux.Get("/", api.Index)
	mux.Post("/signup", api.Signup)
	mux.Post("/recover", api.Recover)
	mux.Post("/verify", api.Verify)
	mux.Get("/user", api.UserGet)
	mux.Put("/user", api.UserUpdate)
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

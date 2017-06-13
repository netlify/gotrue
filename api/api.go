package api

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	"github.com/guregu/kami"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/rs/cors"
)

const (
	audHeaderName  = "X-JWT-AUD"
	defaultVersion = "unknown version"
)

var bearerRegexp = regexp.MustCompile(`^(?:B|b)earer (\S+$)`)

// API is the main REST API
type API struct {
	handler http.Handler
	db      storage.Connection
	mailer  *mailer.Mailer
	config  *conf.Configuration
	version string
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

func (a *API) requestAud(r *http.Request) string {
	p := textproto.MIMEHeader(r.Header)
	if h, exist := p[textproto.CanonicalMIMEHeaderKey(audHeaderName)]; exist && len(h) > 0 {
		return h[0]
	}
	return a.config.JWT.Aud
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a.handler)
}

// NewAPI instantiates a new REST API
func NewAPI(config *conf.Configuration, db storage.Connection, mailer *mailer.Mailer) *API {
	return NewAPIWithVersion(config, db, mailer, defaultVersion)
}

func NewAPIWithVersion(config *conf.Configuration, db storage.Connection, mailer *mailer.Mailer, version string) *API {
	api := &API{config: config, db: db, mailer: mailer, version: version}
	mux := kami.New()

	mux.Use("/user", api.requireAuthentication)
	mux.Use("/logout", api.requireAuthentication)
	mux.Use("/admin/user", api.requireAuthentication)

	mux.Get("/", api.Index)
	mux.Post("/signup", api.Signup)
	mux.Post("/recover", api.Recover)
	mux.Post("/verify", api.Verify)
	mux.Get("/user", api.UserGet)
	mux.Put("/user", api.UserUpdate)
	mux.Post("/token", api.Token)
	mux.Post("/logout", api.Logout)

	// Admin API
	mux.Put("/admin/user", api.adminUserUpdate)
	mux.Post("/admin/user", api.adminUserCreate)
	mux.Delete("/admin/user", api.adminUserDelete)
	mux.Get("/admin/user", api.adminUserGet)

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(mux)
	return api
}

func NewAPIFromConfigFile(filename string, version string) (*API, error) {
	config, err := conf.LoadConfigFile(filename)
	if err != nil {
		return nil, err
	}

	db, err := dial.Dial(config)
	if err != nil {
		return nil, err
	}

	if config.DB.Automigrate {
		if err := db.Automigrate(); err != nil {
			return nil, err
		}
	}

	mailer := mailer.NewMailer(config)
	return NewAPIWithVersion(config, db, mailer, version), nil
}

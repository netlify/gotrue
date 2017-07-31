package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi"
	"github.com/netlify/gotrue/api/provider"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/netlify/netlify-commons/graceful"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
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
	mailer  mailer.Mailer
	config  *conf.Configuration
	version string
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) {
	log := logrus.WithField("component", "api")
	server := graceful.NewGracefulServer(a.handler, log)
	if err := server.Bind(hostAndPort); err != nil {
		log.WithError(err).Fatal("http server bind failed")
	}

	if err := server.Listen(); err != nil {
		log.WithError(err).Fatal("http server listen failed")
	}
}

// NewAPI instantiates a new REST API
func NewAPI(config *conf.Configuration, db storage.Connection, mailer mailer.Mailer) *API {
	return NewAPIWithVersion(context.Background(), config, db, mailer, defaultVersion)
}

// NewAPIWithVersion creates a new REST API using the specified version
func NewAPIWithVersion(ctx context.Context, config *conf.Configuration, db storage.Connection, mailer mailer.Mailer, version string) *API {
	api := &API{config: config, db: db, mailer: mailer, version: version}

	r := newRouter()
	r.Use(addRequestID)
	r.UseBypass(newStructuredLogger(logrus.StandardLogger()))
	r.Use(recoverer)

	r.Get("/", api.Index)
	r.Post("/signup", api.Signup)
	r.Post("/recover", api.Recover)
	r.Post("/verify", api.Verify)
	r.Post("/token", api.Token)
	r.With(api.requireAuthentication).Post("/logout", api.Logout)

	r.Route("/user", func(r *router) {
		r.Use(api.requireAuthentication)
		r.Get("/", api.UserGet)
		r.Put("/", api.UserUpdate)
	})

	r.Route("/admin", func(r *router) {
		r.Use(addGetBody)
		r.Use(api.requireAuthentication)
		r.Use(api.requireAdmin)
		r.Get("/users", api.adminUsers)

		r.Post("/user", api.adminUserCreate)
		r.Get("/user", api.adminUserGet)
		r.Put("/user", api.adminUserUpdate)
		r.Delete("/user", api.adminUserDelete)
	})

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", audHeaderName},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(chi.ServerBaseContext(r, ctx))
	return api
}

// NewAPIFromConfigFile creates a new REST API using the provided configuration file.
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
	return NewAPIWithVersion(context.Background(), config, db, mailer, version), nil
}

// Provider returns a Provider interface for the given name.
func (a *API) Provider(name string) (provider.Provider, error) {
	name = strings.ToLower(name)

	switch name {
	case "github":
		return provider.NewGithubProvider(a.config.External.Github), nil
	case "bitbucket":
		return provider.NewBitbucketProvider(a.config.External.Bitbucket), nil
	case "gitlab":
		return provider.NewGitlabProvider(a.config.External.Gitlab), nil
	default:
		return nil, fmt.Errorf("Provider %s could not be found", name)
	}
}

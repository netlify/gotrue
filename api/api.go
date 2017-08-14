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
	config  *conf.GlobalConfiguration
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
func NewAPI(globalConfig *conf.GlobalConfiguration, db storage.Connection) *API {
	return NewAPIWithVersion(context.Background(), globalConfig, db, defaultVersion)
}

// NewAPIWithVersion creates a new REST API using the specified version
func NewAPIWithVersion(ctx context.Context, globalConfig *conf.GlobalConfiguration, db storage.Connection, version string) *API {
	api := &API{config: globalConfig, db: db, version: version}

	r := newRouter()
	r.Use(addRequestID)
	r.UseBypass(newStructuredLogger(logrus.StandardLogger()))
	r.Use(recoverer)

	r.Route("/", func(r *router) {
		if globalConfig.MultiInstanceMode {
			r.Use(api.loadInstanceConfig)
		}
		r.Get("/health", api.HealthCheck)

		r.Post("/signup", api.Signup)
		r.Post("/invite", api.Invite)
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
	})

	if globalConfig.MultiInstanceMode {
		// Netlify microservice API
		r.With(api.verifyNetlifyRequest).Get("/", api.GetAppManifest)
		r.Route("/instances", func(r *router) {
			r.Use(api.verifyNetlifyRequest)

			r.Post("/", api.CreateInstance)
			r.Route("/{instance_id}", func(r *router) {
				r.With(api.loadInstance)

				r.Get("/", api.GetInstance)
				r.Put("/", api.UpdateInstance)
				r.Delete("/", api.DeleteInstance)
			})
		})
	}

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
	globalConfig, err := conf.LoadGlobalFromFile(filename)
	if err != nil {
		return nil, err
	}
	config, err := conf.LoadConfigFromFile(filename)
	if err != nil {
		return nil, err
	}

	db, err := dial.Dial(globalConfig)
	if err != nil {
		return nil, err
	}

	if globalConfig.DB.Automigrate {
		if err := db.Automigrate(); err != nil {
			return nil, err
		}
	}

	ctx, err := WithInstanceConfig(context.Background(), config, "")
	if err != nil {
		logrus.Fatalf("Error loading instance config: %+v", err)
	}

	return NewAPIWithVersion(ctx, globalConfig, db, version), nil
}

func (a *API) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	return sendJSON(w, http.StatusOK, map[string]string{
		"version":     a.version,
		"name":        "GoTrue",
		"description": "GoTrue is a user registration and authentication API",
	})
}

// Provider returns a Provider interface for the given name.
func (a *API) Provider(ctx context.Context, name string) (provider.Provider, error) {
	config := getConfig(ctx)
	name = strings.ToLower(name)

	switch name {
	case "github":
		return provider.NewGithubProvider(config.External.Github), nil
	case "bitbucket":
		return provider.NewBitbucketProvider(config.External.Bitbucket), nil
	case "gitlab":
		return provider.NewGitlabProvider(config.External.Gitlab), nil
	default:
		return nil, fmt.Errorf("Provider %s could not be found", name)
	}
}

func WithInstanceConfig(ctx context.Context, config *conf.Configuration, instanceID string) (context.Context, error) {
	ctx = withConfig(ctx, config)

	mailer := mailer.NewMailer(config)
	ctx = withMailer(ctx, mailer)
	ctx = withInstanceID(ctx, instanceID)

	return ctx, nil
}

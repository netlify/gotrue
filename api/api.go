package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/imdario/mergo"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/netlify-commons/graceful"
	"github.com/rs/cors"
	"github.com/gofrs/uuid"
	"github.com/sebest/xff"
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
	db      *storage.Connection
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
func NewAPI(globalConfig *conf.GlobalConfiguration, db *storage.Connection) *API {
	return NewAPIWithVersion(context.Background(), globalConfig, db, defaultVersion)
}

// NewAPIWithVersion creates a new REST API using the specified version
func NewAPIWithVersion(ctx context.Context, globalConfig *conf.GlobalConfiguration, db *storage.Connection, version string) *API {
	api := &API{config: globalConfig, db: db, version: version}

	xffmw, _ := xff.Default()
	logger := newStructuredLogger(logrus.StandardLogger())

	r := newRouter()
	r.UseBypass(xffmw.Handler)
	r.Use(addRequestID(globalConfig))
	r.Use(recoverer)

	r.Get("/health", api.HealthCheck)

	r.Route("/callback", func(r *router) {
		r.UseBypass(logger)
		r.Use(api.loadOAuthState)

		if globalConfig.MultiInstanceMode {
			r.Use(api.loadInstanceConfig)
		}
		r.Get("/", api.ExternalProviderCallback)
	})

	r.Route("/", func(r *router) {
		r.UseBypass(logger)

		if globalConfig.MultiInstanceMode {
			r.Use(api.loadJWSSignatureHeader)
			r.Use(api.loadInstanceConfig)
		}

		r.Get("/settings", api.Settings)

		r.Get("/authorize", api.ExternalProviderRedirect)

		r.With(api.requireAdminCredentials).Post("/invite", api.Invite)

		r.With(api.requireEmailProvider).Post("/signup", api.Signup)
		r.With(api.requireEmailProvider).Post("/recover", api.Recover)
		r.With(api.requireEmailProvider).Post("/token", api.Token)
		r.Post("/verify", api.Verify)

		r.With(api.requireAuthentication).Post("/logout", api.Logout)

		r.Route("/user", func(r *router) {
			r.Use(api.requireAuthentication)
			r.Get("/", api.UserGet)
			r.Put("/", api.UserUpdate)
		})

		r.Route("/admin", func(r *router) {
			r.Use(api.requireAdminCredentials)

			r.Route("/audit", func(r *router) {
				r.Get("/", api.adminAuditLog)
			})

			r.Route("/users", func(r *router) {
				r.Get("/", api.adminUsers)
				r.With(api.requireEmailProvider).Post("/", api.adminUserCreate)

				r.Route("/{user_id}", func(r *router) {
					r.Use(api.loadUser)

					r.Get("/", api.adminUserGet)
					r.Put("/", api.adminUserUpdate)
					r.Delete("/", api.adminUserDelete)
				})
			})
		})

		r.Route("/saml", func(r *router) {
			r.Route("/acs", func(r *router) {
				r.Use(api.loadSAMLState)
				r.Post("/", api.ExternalProviderCallback)
			})

			r.Get("/metadata", api.SAMLMetadata)
		})
	})

	if globalConfig.MultiInstanceMode {
		// Operator microservice API
		r.WithBypass(logger).With(api.verifyOperatorRequest).Get("/", api.GetAppManifest)
		r.Route("/instances", func(r *router) {
			r.UseBypass(logger)
			r.Use(api.verifyOperatorRequest)

			r.Post("/", api.CreateInstance)
			r.Route("/{instance_id}", func(r *router) {
				r.Use(api.loadInstance)

				r.Get("/", api.GetInstance)
				r.Put("/", api.UpdateInstance)
				r.Delete("/", api.DeleteInstance)
			})
		})
	}

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", audHeaderName, useCookieHeader},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(chi.ServerBaseContext(r, ctx))
	return api
}

// NewAPIFromConfigFile creates a new REST API using the provided configuration file.
func NewAPIFromConfigFile(filename string, version string) (*API, *conf.Configuration, error) {
	globalConfig, err := conf.LoadGlobal(filename)
	if err != nil {
		return nil, nil, err
	}

	config, err := conf.LoadConfig(filename)
	if err != nil {
		return nil, nil, err
	}

	ctx, err := WithInstanceConfig(context.Background(), config, uuid.Nil)
	if err != nil {
		logrus.Fatalf("Error loading instance config: %+v", err)
	}

	db, err := storage.Dial(globalConfig)
	if err != nil {
		return nil, nil, err
	}

	return NewAPIWithVersion(ctx, globalConfig, db, version), config, nil
}

func (a *API) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	return sendJSON(w, http.StatusOK, map[string]string{
		"version":     a.version,
		"name":        "GoTrue",
		"description": "GoTrue is a user registration and authentication API",
	})
}

func WithInstanceConfig(ctx context.Context, config *conf.Configuration, instanceID uuid.UUID) (context.Context, error) {
	ctx = withConfig(ctx, config)
	ctx = withInstanceID(ctx, instanceID)
	return ctx, nil
}

func (a *API) Mailer(ctx context.Context) mailer.Mailer {
	config := a.getConfig(ctx)
	return mailer.NewMailer(config)
}

func (a *API) getConfig(ctx context.Context) *conf.Configuration {
	obj := ctx.Value(configKey)
	if obj == nil {
		return nil
	}

	config := obj.(*conf.Configuration)

	extConfig := (*a.config).External
	if err := mergo.MergeWithOverwrite(&extConfig, config.External); err != nil {
		return nil
	}
	config.External = extConfig

	smtpConfig := (*a.config).SMTP
	if err := mergo.MergeWithOverwrite(&smtpConfig, config.SMTP); err != nil {
		return nil
	}
	config.SMTP = smtpConfig

	return config
}

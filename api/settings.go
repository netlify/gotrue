package api

import (
	"context"
	"net/http"
)

type ProviderSettings struct {
	Bitbucket bool `json:"bitbucket"`
	GitHub    bool `json:"github"`
	GitLab    bool `json:"gitlab"`
	Google    bool `json:"google"`
	Facebook  bool `json:"facebook"`
	Email     bool `json:"email"`
	SAML      bool `json:"saml"`
}

type ProviderLabels struct {
	SAML string `json:"saml,omitempty"`
}

type Settings struct {
	ExternalProviders ProviderSettings `json:"external"`
	ExternalLabels    ProviderLabels   `json:"external_labels"`
	DisableSignup     bool             `json:"disable_signup"`
	Autoconfirm       bool             `json:"autoconfirm"`
}

func (a *API) handleSettings(w http.ResponseWriter, r *http.Request) error {
	return sendJSON(w, http.StatusOK, a.Settings(r.Context()))
}

func (a *API) Settings(ctx context.Context) *Settings {
	if ctx == nil {
		ctx = a.ctx
	}
	config := a.getConfig(ctx)
	if config == nil {
		return nil
	}
	return &Settings{
		ExternalProviders: ProviderSettings{
			Bitbucket: config.External.Bitbucket.Enabled,
			GitHub:    config.External.Github.Enabled,
			GitLab:    config.External.Gitlab.Enabled,
			Google:    config.External.Google.Enabled,
			Facebook:  config.External.Facebook.Enabled,
			Email:     !config.External.Email.Disabled,
			SAML:      config.External.Saml.Enabled,
		},
		ExternalLabels: ProviderLabels{
			SAML: config.External.Saml.Name,
		},
		DisableSignup: config.DisableSignup,
		Autoconfirm:   config.Mailer.Autoconfirm,
	}
}

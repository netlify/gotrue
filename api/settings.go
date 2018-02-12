package api

import "net/http"

type ProviderSettings struct {
	Bitbucket bool `json:"bitbucket"`
	GitHub    bool `json:"github"`
	GitLab    bool `json:"gitlab"`
	Google    bool `json:"google"`
	Facebook  bool `json:"facebook"`
	Email     bool `json:"email"`
}

type Settings struct {
	ExternalProviders ProviderSettings `json:"external"`
	DisableSignup     bool             `json:"disable_signup"`
	Autoconfirm       bool             `json:"autoconfirm"`
}

func (a *API) Settings(w http.ResponseWriter, r *http.Request) error {
	config := a.getConfig(r.Context())

	return sendJSON(w, http.StatusOK, &Settings{
		ExternalProviders: ProviderSettings{
			Bitbucket: config.External.Bitbucket.Enabled,
			GitHub:    config.External.Github.Enabled,
			GitLab:    config.External.Gitlab.Enabled,
			Google:    config.External.Google.Enabled,
			Facebook:  config.External.Facebook.Enabled,
			Email:     !config.External.Email.Disabled,
		},
		DisableSignup: config.DisableSignup,
		Autoconfirm:   config.Mailer.Autoconfirm,
	})
}

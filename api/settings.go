package api

import "net/http"

type ProviderSettings struct {
	Apple     bool `json:"apple"`
	Bitbucket bool `json:"bitbucket"`
	GitHub    bool `json:"github"`
	GitLab    bool `json:"gitlab"`
	Google    bool `json:"google"`
	Facebook  bool `json:"facebook"`
	Twitter   bool `json:"twitter"`
	Azure     bool `json:"azure"`
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

func (a *API) Settings(w http.ResponseWriter, r *http.Request) error {
	config := a.getConfig(r.Context())

	return sendJSON(w, http.StatusOK, &Settings{
		ExternalProviders: ProviderSettings{
			Apple:     config.External.Apple.Enabled,
			Bitbucket: config.External.Bitbucket.Enabled,
			GitHub:    config.External.Github.Enabled,
			GitLab:    config.External.Gitlab.Enabled,
			Google:    config.External.Google.Enabled,
			Facebook:  config.External.Facebook.Enabled,
			Twitter:   config.External.Twitter.Enabled,
			Azure:     config.External.Azure.Enabled,
			Email:     !config.External.Email.Disabled,
			SAML:      config.External.Saml.Enabled,
		},
		ExternalLabels: ProviderLabels{
			SAML: config.External.Saml.Name,
		},
		DisableSignup: config.DisableSignup,
		Autoconfirm:   config.Mailer.Autoconfirm,
	})
}

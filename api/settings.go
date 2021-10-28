package api

import "net/http"

type ProviderSettings struct {
	Apple     bool `json:"apple"`
	Azure     bool `json:"azure"`
	Bitbucket bool `json:"bitbucket"`
	Discord   bool `json:"discord"`
	GitHub    bool `json:"github"`
	GitLab    bool `json:"gitlab"`
	Google    bool `json:"google"`
	Facebook  bool `json:"facebook"`
	Spotify   bool `json:"spotify"`
	Slack     bool `json:"slack"`
	Twitch    bool `json:"twitch"`
	Twitter   bool `json:"twitter"`
	Email     bool `json:"email"`
	Phone     bool `json:"phone"`
	SAML      bool `json:"saml"`
}

type ProviderLabels struct {
	SAML string `json:"saml,omitempty"`
}

type Settings struct {
	ExternalProviders ProviderSettings `json:"external"`
	ExternalLabels    ProviderLabels   `json:"external_labels"`
	DisableSignup     bool             `json:"disable_signup"`
	MailerAutoconfirm bool             `json:"mailer_autoconfirm"`
	PhoneAutoconfirm  bool             `json:"phone_autoconfirm"`
	SmsProvider       string           `json:"sms_provider"`
}

func (a *API) Settings(w http.ResponseWriter, r *http.Request) error {
	config := a.getConfig(r.Context())

	return sendJSON(w, http.StatusOK, &Settings{
		ExternalProviders: ProviderSettings{
			Apple:     config.External.Apple.Enabled,
			Azure:     config.External.Azure.Enabled,
			Bitbucket: config.External.Bitbucket.Enabled,
			Discord:   config.External.Discord.Enabled,
			GitHub:    config.External.Github.Enabled,
			GitLab:    config.External.Gitlab.Enabled,
			Google:    config.External.Google.Enabled,
			Facebook:  config.External.Facebook.Enabled,
			Spotify:   config.External.Spotify.Enabled,
			Slack:     config.External.Slack.Enabled,
			Twitch:    config.External.Twitch.Enabled,
			Twitter:   config.External.Twitter.Enabled,
			Email:     config.External.Email.Enabled,
			Phone:     config.External.Phone.Enabled,
			SAML:      config.External.Saml.Enabled,
		},
		ExternalLabels: ProviderLabels{
			SAML: config.External.Saml.Name,
		},

		DisableSignup:     config.DisableSignup,
		MailerAutoconfirm: config.Mailer.Autoconfirm,
		PhoneAutoconfirm:  config.Sms.Autoconfirm,
		SmsProvider:       config.Sms.Provider,
	})
}

package sms_provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/netlify/gotrue/conf"
)

const (
	defaultMessagebirdApiBase = "https://rest.messagebird.com"
)

type MessagebirdProvider struct {
	Config  *conf.MessagebirdProviderConfiguration
	APIPath string
}

type MessagebirdResponseRecipients struct {
	TotalSentCount int `json:"totalSentCount"`
}

type MessagebirdResponse struct {
	Recipients MessagebirdResponseRecipients `json:"recipients"`
}

type MessagebirdError struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter"`
}

type MessagebirdErrResponse struct {
	Errors []MessagebirdError `json:"errors"`
}

func (t MessagebirdErrResponse) Error() string {
	return t.Errors[0].Description
}

// Creates a SmsProvider with the Messagebird Config
func NewMessagebirdProvider(config conf.MessagebirdProviderConfiguration) (SmsProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	apiPath := defaultMessagebirdApiBase + "/messages"
	return &MessagebirdProvider{
		Config:  &config,
		APIPath: apiPath,
	}, nil
}

// Send an SMS containing the OTP with Messagebird's API
func (t MessagebirdProvider) SendSms(phone string, message string) error {
	body := url.Values{
		"originator": {t.Config.Originator},
		"body":       {message},
		"recipients": {phone},
		"type":       {"sms"},
		"datacoding": {"unicode"},
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", t.APIPath, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Authorization", "AccessKey "+t.Config.AccessKey)
	res, err := client.Do(r)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusUnprocessableEntity {
		resp := &MessagebirdErrResponse{}
		if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
			return err
		}
		return resp
	}
	defer res.Body.Close()

	// validate sms status
	resp := &MessagebirdResponse{}
	derr := json.NewDecoder(res.Body).Decode(resp)
	if derr != nil {
		return derr
	}

	if resp.Recipients.TotalSentCount == 0 {
		return fmt.Errorf("Messagebird error: total sent count is 0")
	}

	return nil
}

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
	defaultTwilioApiBase = "https://api.twilio.com"
	apiVersion           = "2010-04-01"
)

type TwilioProvider struct {
	Config  *conf.TwilioProviderConfiguration
	APIPath string
}

type SmsStatus struct {
	To           string `json:"to"`
	From         string `json:"from"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Body         string `json:"body"`
}

// Creates a SmsProvider with the Twilio Config
func NewTwilioProvider(config conf.TwilioProviderConfiguration) (SmsProvider, error) {
	apiPath := defaultTwilioApiBase + "/" + apiVersion + "/" + "Accounts" + "/" + config.AccountSid + "/Messages.json"
	return &TwilioProvider{
		Config:  &config,
		APIPath: apiPath,
	}, nil
}

// Send an SMS containing the OTP with Twilio's API
func (t TwilioProvider) SendSms(phone string, otp string) error {
	body := url.Values{
		"To":      {"+" + phone}, // twilio api requires "+" extension to be included
		"Channel": {"sms"},
		"From":    {t.Config.MessageServiceSid},
		"Body":    {otp},
	}

	client := &http.Client{}
	r, err := http.NewRequest("POST", t.APIPath, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(t.Config.AccountSid, t.Config.AuthToken)
	res, err := client.Do(r)
	if res.StatusCode == http.StatusBadRequest || err != nil {
		return fmt.Errorf("Bad request: %v", err)
	}
	defer res.Body.Close()

	// validate sms status
	resp := &SmsStatus{}
	derr := json.NewDecoder(res.Body).Decode(resp)
	if derr != nil {
		return derr
	}

	if resp.Status == "failed" || resp.Status == "undelivered" {
		return fmt.Errorf("Twilio error: %v %v", resp.ErrorMessage, resp.ErrorCode)
	}

	return nil
}

package sms_provider

import (
	"fmt"

	"github.com/netlify/gotrue/conf"
)

type SmsProvider interface {
	SendSms(phone, otp string) error
}

func GetSmsProvider(config conf.Configuration) (SmsProvider, error) {
	switch name := config.Sms.Provider; name {
	case "twilio":
		return NewTwilioProvider(config.Sms.Twilio)
	default:
		return nil, fmt.Errorf("Sms Provider %s could not be found", name)
	}
}

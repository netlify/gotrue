package metering

import (
	"net/http"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var logger = logrus.StandardLogger().WithField("metering", true)

func RecordLogin(r *http.Request, loginType string, userID, instanceID uuid.UUID) {
	logger.WithFields(logrus.Fields{
		"action":      "login",
		"domain":      r.Host,
		"instance_id": instanceID.String(),
		"user_id":     userID.String(),
	}).Info("Login")
}

package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/gobuffalo/uuid"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
)

type HookEvent string

const (
	headerHookSignature = "x-webhook-signature"
	defaultHookRetries  = 3
	gotrueIssuer        = "gotrue"
	ValidateEvent       = "validate"
	SignupEvent         = "signup"
	LoginEvent          = "login"
)

var defaultTimeout = time.Second * 5

type webhookClaims struct {
	jwt.StandardClaims
	SHA256 string `json:"sha256"`
}

type Webhook struct {
	*conf.WebhookConfig

	instanceID uuid.UUID
	jwtSecret  string
	claims     jwt.Claims
	payload    []byte
	headers    map[string]string
}

type WebhookResponse struct {
	AppMetaData  map[string]interface{} `json:"app_metadata,omitempty"`
	UserMetaData map[string]interface{} `json:"user_metadata,omitempty"`
}

func (w *Webhook) trigger() (io.ReadCloser, error) {
	timeout := defaultTimeout
	if w.TimeoutSec > 0 {
		timeout = time.Duration(w.TimeoutSec) * time.Second
	}

	if w.Retries == 0 {
		w.Retries = defaultHookRetries
	}

	hooklog := logrus.WithFields(logrus.Fields{
		"component":   "webhook",
		"url":         w.URL,
		"signed":      w.jwtSecret != "",
		"instance_id": w.instanceID,
	})
	client := http.Client{
		Timeout: timeout,
	}
	client.Transport = SafeRoundtripper(client.Transport, hooklog)

	for i := 0; i < w.Retries; i++ {
		hooklog = hooklog.WithField("attempt", i+1)
		hooklog.Info("Starting to perform signup hook request")

		req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewBuffer(w.payload))
		if err != nil {
			return nil, internalServerError("Failed to make request object").WithInternalError(err)
		}
		req.Header.Set("Content-Type", "application/json")
		watcher, req := watchForConnection(req)

		if w.jwtSecret != "" {
			header, jwtErr := w.generateSignature()
			if jwtErr != nil {
				return nil, jwtErr
			}
			req.Header.Set(headerHookSignature, header)
		}

		start := time.Now()
		rsp, err := client.Do(req)
		if err != nil {
			if terr, ok := err.(net.Error); ok && terr.Timeout() {
				// timed out - try again?
				if i == w.Retries-1 {
					closeBody(rsp)
					return nil, httpError(http.StatusGatewayTimeout, "Failed to perform webhook in time frame (%v seconds)", timeout.Seconds())
				}
				hooklog.Info("Request timed out")
				continue
			} else if watcher.gotConn {
				closeBody(rsp)
				return nil, internalServerError("Failed to trigger webhook to %s", w.URL).WithInternalError(err)
			} else {
				closeBody(rsp)
				return nil, httpError(http.StatusBadGateway, "Failed to connect to %s", w.URL)
			}
		}
		dur := time.Since(start)
		rspLog := hooklog.WithFields(logrus.Fields{
			"status_code": rsp.StatusCode,
			"dur":         dur.Nanoseconds(),
		})
		switch rsp.StatusCode {
		case http.StatusOK, http.StatusNoContent, http.StatusAccepted:
			rspLog.Infof("Finished processing webhook in %s", dur)
			var body io.ReadCloser
			if rsp.ContentLength > 0 {
				body = rsp.Body
			}
			return body, nil
		default:
			rspLog.Infof("Bad response for webhook %d in %s", rsp.StatusCode, dur)
		}
	}

	hooklog.Infof("Failed to process webhook for %s after %d attempts", w.URL, w.Retries)
	return nil, unprocessableEntityError("Failed to handle signup webhook")
}

func (w *Webhook) generateSignature() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, w.claims)
	tokenString, err := token.SignedString([]byte(w.jwtSecret))
	if err != nil {
		return "", internalServerError("Failed build signing string").WithInternalError(err)
	}
	return tokenString, nil
}

func closeBody(rsp *http.Response) {
	if rsp != nil && rsp.Body != nil {
		rsp.Body.Close()
	}
}

func triggerEventHooks(ctx context.Context, conn *storage.Connection, event HookEvent, user *models.User, instanceID uuid.UUID, config *conf.Configuration) error {
	if config.Webhook.URL != "" {
		hookURL, err := url.Parse(config.Webhook.URL)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse Webhook URL")
		}
		if !config.Webhook.HasEvent(string(event)) {
			return nil
		}
		return triggerHook(ctx, hookURL, config.Webhook.Secret, conn, event, user, instanceID, config)
	}

	fun := getFunctionHooks(ctx)
	if fun == nil {
		return nil
	}

	for _, eventHookURL := range fun[string(event)] {
		hookURL, err := url.Parse(eventHookURL)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse Event Function Hook URL")
		}
		err = triggerHook(ctx, hookURL, config.JWT.Secret, conn, event, user, instanceID, config)
		if err != nil {
			return err
		}
	}
	return nil
}

func triggerHook(ctx context.Context, hookURL *url.URL, secret string, conn *storage.Connection, event HookEvent, user *models.User, instanceID uuid.UUID, config *conf.Configuration) error {
	if !hookURL.IsAbs() {
		siteURL, err := url.Parse(config.SiteURL)
		if err != nil {
			return errors.Wrapf(err, "Failed to parse Site URL")
		}
		hookURL.Scheme = siteURL.Scheme
		hookURL.Host = siteURL.Host
		hookURL.User = siteURL.User
	}

	payload := struct {
		Event      HookEvent    `json:"event"`
		InstanceID uuid.UUID    `json:"instance_id,omitempty"`
		User       *models.User `json:"user"`
	}{
		Event:      event,
		InstanceID: instanceID,
		User:       user,
	}
	data, err := json.Marshal(&payload)
	if err != nil {
		return internalServerError("Failed to serialize the data for signup webhook").WithInternalError(err)
	}

	sha, err := checksum(data)
	if err != nil {
		return internalServerError("Failed to checksum the data for signup webhook").WithInternalError(err)
	}

	claims := webhookClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt: time.Now().Unix(),
			Subject:  instanceID.String(),
			Issuer:   gotrueIssuer,
		},
		SHA256: sha,
	}

	w := Webhook{
		WebhookConfig: &config.Webhook,
		jwtSecret:     secret,
		instanceID:    instanceID,
		claims:        claims,
		payload:       data,
	}

	w.URL = hookURL.String()

	body, err := w.trigger()
	defer func() {
		if body != nil {
			body.Close()
		}
	}()
	if err == nil && body != nil {
		webhookRsp := &WebhookResponse{}
		decoder := json.NewDecoder(body)
		if err = decoder.Decode(webhookRsp); err != nil {
			return internalServerError("Webhook returned malformed JSON: %v", err).WithInternalError(err)
		}
		return conn.Transaction(func(tx *storage.Connection) error {
			if webhookRsp.UserMetaData != nil {
				user.UserMetaData = nil
				if terr := user.UpdateUserMetaData(tx, webhookRsp.UserMetaData); terr != nil {
					return terr
				}
			}
			if webhookRsp.AppMetaData != nil {
				user.AppMetaData = nil
				if terr := user.UpdateAppMetaData(tx, webhookRsp.AppMetaData); terr != nil {
					return terr
				}
			}
			return nil
		})
	}
	return err
}

func watchForConnection(req *http.Request) (*connectionWatcher, *http.Request) {
	w := new(connectionWatcher)
	t := &httptrace.ClientTrace{
		GotConn: w.GotConn,
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), t))
	return w, req
}

func checksum(data []byte) (string, error) {
	sha := sha256.New()
	_, err := sha.Write(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(sha.Sum(nil)), nil
}

type connectionWatcher struct {
	gotConn bool
}

func (c *connectionWatcher) GotConn(_ httptrace.GotConnInfo) {
	c.gotConn = true
}

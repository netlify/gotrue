package storage

import (
	"errors"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/kelseyhightower/envconfig"
)

var sessionName = "_gotrue_session"
var Store sessions.Store

type SessionConfig struct {
	Key []byte `envconfig:"GOTRUE_SESSION_KEY"`
}

func init() {
	var sessionConfig SessionConfig
	err := envconfig.Process("GOTRUE_SESSION_KEY", &sessionConfig)
	if err != nil || len(sessionConfig.Key) == 0 {
		sessionConfig.Key = securecookie.GenerateRandomKey(32)
	}
	Store = sessions.NewCookieStore(sessionConfig.Key)
}

func StoreInSession(key string, value string, req *http.Request, res http.ResponseWriter) error {
	session, _ := Store.New(req, sessionName)
	session.Values[key] = value
	return session.Save(req, res)
}

func GetFromSession(key string, req *http.Request) (string, error) {
	session, _ := Store.Get(req, sessionName)
	value, ok := session.Values[key]
	if !ok {
		return "", errors.New("session could not be found for this request")
	}

	return value.(string), nil
}

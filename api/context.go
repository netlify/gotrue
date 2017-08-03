package api

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
)

type contextKey string

func (c contextKey) String() string {
	return "gotrue api context key " + string(c)
}

const (
	tokenKey      = contextKey("jwt")
	requestIDKey  = contextKey("request_id")
	configKey     = contextKey("config")
	mailerKey     = contextKey("mailer")
	instanceIDKey = contextKey("instance_id")
	instanceKey   = contextKey("instance")
)

// WithToken adds the JWT token to the context.
func withToken(ctx context.Context, token *jwt.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// GetToken reads the JWT token from the context.
func getToken(ctx context.Context) *jwt.Token {
	obj := ctx.Value(tokenKey)
	if obj == nil {
		return nil
	}

	return obj.(*jwt.Token)
}

// withRequestID adds the provided request ID to the context.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// getRequestID reads the request ID from the context.
func getRequestID(ctx context.Context) string {
	obj := ctx.Value(requestIDKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// withConfig adds the tenant configuration to the context.
func withConfig(ctx context.Context, config *conf.Configuration) context.Context {
	return context.WithValue(ctx, configKey, config)
}

// getConfig reads the tenant configuration from the context.
func getConfig(ctx context.Context) *conf.Configuration {
	obj := ctx.Value(configKey)
	if obj == nil {
		return nil
	}

	return obj.(*conf.Configuration)
}

// withMailer adds the mailer to the context.
func withMailer(ctx context.Context, mailer mailer.Mailer) context.Context {
	return context.WithValue(ctx, mailerKey, mailer)
}

// getMailer reads the mailer from the context.
func getMailer(ctx context.Context) mailer.Mailer {
	obj := ctx.Value(mailerKey)
	if obj == nil {
		return nil
	}
	return obj.(mailer.Mailer)
}

// withInstanceID adds the instance id to the context.
func withInstanceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, instanceIDKey, id)
}

// getInstanceID reads the instance id from the context.
func getInstanceID(ctx context.Context) string {
	obj := ctx.Value(instanceIDKey)
	if obj == nil {
		return ""
	}
	return obj.(string)
}

// withInstance adds the instance id to the context.
func withInstance(ctx context.Context, i *models.Instance) context.Context {
	return context.WithValue(ctx, instanceKey, i)
}

// getInstance reads the instance id from the context.
func getInstance(ctx context.Context) *models.Instance {
	obj := ctx.Value(instanceKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.Instance)
}

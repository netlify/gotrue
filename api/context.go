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
	tokenKey                = contextKey("jwt")
	requestIDKey            = contextKey("request_id")
	configKey               = contextKey("config")
	mailerKey               = contextKey("mailer")
	inviteTokenKey          = contextKey("invite_token")
	instanceIDKey           = contextKey("instance_id")
	instanceKey             = contextKey("instance")
	signatureKey            = contextKey("signature")
	netlifyIDKey            = contextKey("netlify_id")
	externalProviderTypeKey = contextKey("external_provider_type")
	userKey                 = contextKey("user")
)

// withToken adds the JWT token to the context.
func withToken(ctx context.Context, token *jwt.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// getToken reads the JWT token from the context.
func getToken(ctx context.Context) *jwt.Token {
	obj := ctx.Value(tokenKey)
	if obj == nil {
		return nil
	}

	return obj.(*jwt.Token)
}

func getClaims(ctx context.Context) *GoTrueClaims {
	token := getToken(ctx)
	if token == nil {
		return nil
	}
	return token.Claims.(*GoTrueClaims)
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

// withMailer adds the mailer to the context.
func withMailer(ctx context.Context, mailer mailer.Mailer) context.Context {
	return context.WithValue(ctx, mailerKey, mailer)
}

func getConfig(ctx context.Context) *conf.Configuration {
	obj := ctx.Value(configKey)
	if obj == nil {
		return nil
	}
	return obj.(*conf.Configuration)
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

// withUser adds the user id to the context.
func withUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// getUser reads the user id from the context.
func getUser(ctx context.Context) *models.User {
	obj := ctx.Value(userKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.User)
}

// withSignature adds the provided request ID to the context.
func withSignature(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, signatureKey, id)
}

// getSignature reads the request ID from the context.
func getSignature(ctx context.Context) string {
	obj := ctx.Value(signatureKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// withNetlifyID adds the provided request ID to the context.
func withNetlifyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, netlifyIDKey, id)
}

// getNetlifyID reads the request ID from the context.
func getNetlifyID(ctx context.Context) string {
	obj := ctx.Value(netlifyIDKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

func withInviteToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, inviteTokenKey, token)
}

func getInviteToken(ctx context.Context) string {
	obj := ctx.Value(inviteTokenKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// withExternalProviderType adds the provided request ID to the context.
func withExternalProviderType(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, externalProviderTypeKey, id)
}

// getExternalProviderType reads the request ID from the context.
func getExternalProviderType(ctx context.Context) string {
	obj := ctx.Value(externalProviderTypeKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

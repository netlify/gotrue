package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/api/provider"
	"github.com/netlify/gotrue/models"
	"github.com/sirupsen/logrus"
)

type ExternalProviderClaims struct {
	NetlifyMicroserviceClaims
	Provider    string `json:"provider"`
	InviteToken string `json:"invite_token,omitempty"`
}

// SignupParams are the parameters the Signup endpoint accepts
type ExternalSignupParams struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
}

func (a *API) ExternalProviderRedirect(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	providerType := r.URL.Query().Get("provider")
	provider, err := a.Provider(ctx, providerType)
	if err != nil {
		return badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	inviteToken := r.URL.Query().Get("invite_token")
	if inviteToken != "" {
		_, userErr := a.db.FindUserByConfirmationToken(inviteToken)
		if userErr != nil {
			if models.IsNotFoundError(userErr) {
				return notFoundError(userErr.Error())
			}
			return internalServerError("Database error finding user").WithInternalError(userErr)
		}
	}

	log := getLogEntry(r)
	log.WithField("provider", providerType).Info("Redirecting to external provider")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, ExternalProviderClaims{
		NetlifyMicroserviceClaims: NetlifyMicroserviceClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			},
			SiteURL:    config.SiteURL,
			InstanceID: getInstanceID(ctx),
			NetlifyID:  getNetlifyID(ctx),
		},
		Provider:    providerType,
		InviteToken: inviteToken,
	})
	tokenString, err := token.SignedString([]byte(a.config.OperatorToken))
	if err != nil {
		return internalServerError("Error creating state").WithInternalError(err)
	}

	http.Redirect(w, r, provider.AuthCodeURL(tokenString), http.StatusFound)
	return nil
}

func (a *API) ExternalProviderCallback(w http.ResponseWriter, r *http.Request) error {
	a.redirectErrors(a.internalExternalProviderCallback, w, r)
	return nil
}

func (a *API) internalExternalProviderCallback(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)
	instanceID := getInstanceID(ctx)
	rq := r.URL.Query()

	extError := rq.Get("error")
	if extError != "" {
		return oauthError(extError, rq.Get("error_description"))
	}

	oauthCode := rq.Get("code")
	if oauthCode == "" {
		return badRequestError("Authorization code missing")
	}

	providerType := getExternalProviderType(ctx)
	provider, err := a.Provider(ctx, providerType)
	if err != nil {
		return badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	log := getLogEntry(r)
	log.WithFields(logrus.Fields{
		"provider": providerType,
		"code":     oauthCode,
	}).Debug("Exchanging oauth code")

	tok, err := provider.GetOAuthToken(oauthCode)
	if err != nil {
		return internalServerError("Unable to exchange external code: %s", oauthCode).WithInternalError(err)
	}

	aud := a.requestAud(ctx, r)
	userData, err := provider.GetUserData(ctx, tok)
	if err != nil {
		return internalServerError("Error getting user email from external provider").WithInternalError(err)
	}

	var user *models.User
	inviteToken := getInviteToken(ctx)
	if inviteToken != "" {
		user, err = a.db.FindUserByConfirmationToken(inviteToken)
		if err != nil {
			if models.IsNotFoundError(err) {
				return notFoundError(err.Error())
			}
			return internalServerError("Database error finding user").WithInternalError(err)
		}

		if user.Email != userData.Email {
			return badRequestError("Invited email does not match email from external provider").WithInternalMessage("invited=%s external=%s", user.Email, userData.Email)
		}

		if user.AppMetaData == nil {
			user.AppMetaData = make(map[string]interface{})
		}
		user.AppMetaData["provider"] = providerType
		if user.UserMetaData == nil {
			user.UserMetaData = make(map[string]interface{})
		}
		for k, v := range userData.Metadata {
			if v != "" {
				user.UserMetaData[k] = v
			}
		}

		if config.Webhook.HasEvent("signup") {
			if err := triggerHook(SignupEvent, user, instanceID, config); err != nil {
				return err
			}
			a.db.UpdateUser(user)
		}

		// confirm because they were able to respond to invite email
		user.Confirm()
	} else {
		params := &SignupParams{
			Provider: providerType,
			Email:    userData.Email,
		}
		if params.Data == nil {
			params.Data = make(map[string]interface{})
		}
		for k, v := range userData.Metadata {
			if v != "" {
				params.Data[k] = v
			}
		}

		user, err = a.db.FindUserByEmailAndAudience(instanceID, params.Email, aud)
		if err != nil && !models.IsNotFoundError(err) {
			return internalServerError("Error checking for duplicate users").WithInternalError(err)
		}
		if user == nil {
			if config.DisableSignup {
				return forbiddenError("Signups not allowed for this instance")
			}

			user, err = a.signupNewUser(ctx, params, aud)
			if err != nil {
				return err
			}
		}

		if !user.IsConfirmed() {
			if !userData.Verified && !config.Mailer.Autoconfirm {
				mailer := getMailer(ctx)
				if confirmationErr := mailer.ConfirmationMail(user); confirmationErr != nil {
					return internalServerError("Error sending confirmation mail").WithInternalError(confirmationErr)
				}
				now := time.Now()
				user.ConfirmationSentAt = &now

				if confirmationErr := a.db.UpdateUser(user); confirmationErr != nil {
					return internalServerError("Error updating user in database").WithInternalError(confirmationErr)
				}
				// email must be verified to issue a token
				http.Redirect(w, r, config.External.RedirectURL, http.StatusFound)
			}

			if config.Webhook.HasEvent("signup") {
				if err := triggerHook(SignupEvent, user, instanceID, config); err != nil {
					return err
				}
				a.db.UpdateUser(user)
			}

			// fall through to auto-confirm and issue token
			user.Confirm()
		} else {
			if config.Webhook.HasEvent("login") {
				if err := triggerHook(LoginEvent, user, instanceID, config); err != nil {
					return err
				}
				a.db.UpdateUser(user)
			}
		}
	}

	token, err := a.issueRefreshToken(ctx, user)
	if err != nil {
		return oauthError("server_error", err.Error())
	}
	q := url.Values{}
	q.Set("access_token", token.Token)
	q.Set("token_type", token.TokenType)
	q.Set("expires_in", strconv.Itoa(token.ExpiresIn))
	q.Set("refresh_token", token.RefreshToken)

	http.Redirect(w, r, config.External.RedirectURL+"#"+q.Encode(), http.StatusFound)
	return nil
}

// loadOAuthState parses the `state` query parameter as a JWS payload,
// extracting the provider requested
func (a *API) loadOAuthState(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	if state == "" {
		return nil, badRequestError("OAuth state parameter missing")
	}

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err := p.ParseWithClaims(state, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.OperatorToken), nil
	})
	if err != nil || claims.Provider == "" {
		return nil, badRequestError("OAuth state is invalid: %v", err)
	}
	if claims.InviteToken != "" {
		ctx = withInviteToken(ctx, claims.InviteToken)
	}

	ctx = withExternalProviderType(ctx, claims.Provider)
	return withSignature(ctx, state), nil
}

// Provider returns a Provider interface for the given name.
func (a *API) Provider(ctx context.Context, name string) (provider.Provider, error) {
	config := a.getConfig(ctx)
	name = strings.ToLower(name)

	switch name {
	case "bitbucket":
		return provider.NewBitbucketProvider(config.External.Bitbucket)
	case "github":
		return provider.NewGithubProvider(config.External.Github)
	case "gitlab":
		return provider.NewGitlabProvider(config.External.Gitlab)
	case "google":
		return provider.NewGoogleProvider(config.External.Google)
	case "facebook":
		return provider.NewFacebookProvider(config.External.Facebook)
	default:
		return nil, fmt.Errorf("Provider %s could not be found", name)
	}
}

func (a *API) redirectErrors(handler apiHandler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	config := a.getConfig(ctx)
	log := getLogEntry(r)
	errorID := getRequestID(ctx)
	err := handler(w, r)
	if err != nil {
		q := url.Values{}
		switch e := err.(type) {
		case *HTTPError:
			if str, ok := oauthErrorMap[e.Code]; ok {
				q.Set("error", str)
			} else {
				q.Set("error", "server_error")
			}
			if e.Code >= http.StatusInternalServerError {
				e.ErrorID = errorID
				// this will get us the stack trace too
				log.WithError(e.Cause()).Error(e.Error())
			} else {
				log.WithError(e.Cause()).Info(e.Error())
			}
			q.Set("error_description", e.Message)
		case *OAuthError:
			q.Set("error", e.Err)
			q.Set("error_description", e.Description)
			log.WithError(e.Cause()).Info(e.Error())
		default:
			q.Set("error", "server_error")
			q.Set("error_description", err.Error())
		}
		http.Redirect(w, r, config.External.RedirectURL+"#"+q.Encode(), http.StatusFound)
	}
}

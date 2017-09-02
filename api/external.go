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
)

type ExternalProviderClaims struct {
	NetlifyMicroserviceClaims
	Provider string `json:"provider"`
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
		return badRequestError("Unsupported provider: %+v", err)
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
		Provider: providerType,
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
		return badRequestError("Unsupported provider: %+v", err)
	}

	tok, err := provider.GetOAuthToken(oauthCode)
	if err != nil {
		return internalServerError("Unable to exchange external code: %s", oauthCode).WithInternalError(err)
	}

	aud := a.requestAud(ctx, r)
	userData, err := provider.GetUserData(ctx, tok)
	if err != nil {
		return internalServerError("Error getting user email from external provider").WithInternalError(err)
	}

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

	user, err := a.db.FindUserByEmailAndAudience(instanceID, params.Email, aud)
	if err != nil && !models.IsNotFoundError(err) {
		return internalServerError("Error checking for duplicate users").WithInternalError(err)
	}
	if user == nil {
		user, err = a.signupNewUser(ctx, params, aud)
		if err != nil {
			return err
		}
	}

	if !user.IsConfirmed() {
		if !userData.Verified && !config.Mailer.Autoconfirm {
			mailer := getMailer(ctx)
			if err := mailer.ConfirmationMail(user); err != nil {
				return internalServerError("Error sending confirmation mail").WithInternalError(err)
			}
			now := time.Now()
			user.ConfirmationSentAt = &now

			if err := a.db.UpdateUser(user); err != nil {
				return internalServerError("Error updating user in database").WithInternalError(err)
			}
			// email must be verified to issue a token
			http.Redirect(w, r, config.External.RedirectURL, http.StatusFound)
		}

		// fall through to auto-confirm and issue token
		user.Confirm()
	}

	now := time.Now()
	user.LastSignInAt = &now
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

	ctx = withExternalProviderType(ctx, claims.Provider)
	return withSignature(ctx, state), nil
}

// Provider returns a Provider interface for the given name.
func (a *API) Provider(ctx context.Context, name string) (provider.Provider, error) {
	config := a.getConfig(ctx)
	name = strings.ToLower(name)

	switch name {
	case "bitbucket":
		return provider.NewBitbucketProvider(config.External.Bitbucket), nil
	case "github":
		return provider.NewGithubProvider(config.External.Github), nil
	case "gitlab":
		return provider.NewGitlabProvider(config.External.Gitlab), nil
	case "google":
		return provider.NewGoogleProvider(config.External.Google), nil
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
			q.Set("error_description", err.Error())
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

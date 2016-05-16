package api

import (
	"net/http"

	"golang.org/x/net/context"
)

func (a *API) Verify(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Find user by ConfirmationToken

	// If alreay confirmed. Deny
	// If

}

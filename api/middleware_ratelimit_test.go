package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netlify/gotrue/conf"
)

func TestRateLimitKey(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		headerVal  string
		remoteAddr string
		want       string
	}{
		{"configured header present", "X-Forwarded-For", "1.2.3.4", "10.0.0.1:54321", "1.2.3.4"},
		{"configured header empty falls back to ip", "X-Forwarded-For", "", "10.0.0.1:54321", "10.0.0.1"},
		{"no header configured falls back to ip", "", "", "10.0.0.1:54321", "10.0.0.1"},
		{"remoteAddr without port falls back as-is", "", "", "10.0.0.1", "10.0.0.1"},
		{"ipv6 remoteAddr", "", "", "[::1]:12345", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &conf.GlobalConfiguration{}
			config.RateLimitHeader = tt.headerName
			a := &API{config: config}

			req := httptest.NewRequest("POST", "/token", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerName != "" && tt.headerVal != "" {
				req.Header.Set(tt.headerName, tt.headerVal)
			}

			assert.Equal(t, tt.want, a.rateLimitKey(req))
		})
	}
}

package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitBodySize(t *testing.T) {
	const limit int64 = 16

	tests := []struct {
		name     string
		body     string
		wantErr  bool
		wantRead string
	}{
		{"under limit reads fully", "small body", false, "small body"},
		{"at limit reads fully", strings.Repeat("a", int(limit)), false, strings.Repeat("a", int(limit))},
		{"over limit errors", strings.Repeat("a", int(limit)+1), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				readErr  error
				readBody []byte
			)

			handler := limitBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				readBody, readErr = io.ReadAll(r.Body)
			}))

			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if tt.wantErr {
				require.Error(t, readErr)
				var mbe *http.MaxBytesError
				assert.True(t, errors.As(readErr, &mbe), "expected *http.MaxBytesError, got %T", readErr)
				return
			}
			require.NoError(t, readErr)
			assert.Equal(t, tt.wantRead, string(readBody))
		})
	}
}

func TestLimitBodySizeNilBodyIsSafe(t *testing.T) {
	handler := limitBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

package api

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginate(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantPage    uint64
		wantPerPage uint64
		wantErr     bool
	}{
		{"defaults", "", 1, defaultPerPage, false},
		{"explicit page", "page=3", 3, defaultPerPage, false},
		{"explicit per_page", "per_page=10", 1, 10, false},
		{"per_page at max", fmt.Sprintf("per_page=%d", maxPerPage), 1, maxPerPage, false},
		{"per_page above max", fmt.Sprintf("per_page=%d", maxPerPage+1), 0, 0, true},
		{"per_page huge", "per_page=999999999", 0, 0, true},
		{"page not a number", "page=abc", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin/users?"+tt.query, nil)
			p, err := paginate(req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantPerPage, p.PerPage)
		})
	}
}

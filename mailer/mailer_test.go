package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSiteURL(t *testing.T) {
	cases := []struct {
		SiteURL  string
		Path     string
		Fragment string
		Expected string
	}{
		{"https://test.example.com", "/templates/confirm.html", "", "https://test.example.com/templates/confirm.html"},
		{"https://test.example.com/removedpath", "/templates/confirm.html", "", "https://test.example.com/templates/confirm.html"},
		{"https://test.example.com/", "/trailingslash/", "", "https://test.example.com/trailingslash/"},
		{"https://test.example.com", "f", "fragment", "https://test.example.com/f#fragment"},
	}

	for _, c := range cases {
		act, err := getSiteURL(c.SiteURL, c.Path, c.Fragment)
		assert.NoError(t, err, c.Expected)
		assert.Equal(t, c.Expected, act)
	}
}

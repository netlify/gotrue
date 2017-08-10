package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSiteURL(t *testing.T) {
	cases := []struct {
		SiteURL  string
		Folder   string
		Filename string
		Expected string
	}{
		{"https://test.example.com", "/.netlify/gotrue/templates", "confirm.html", "https://test.example.com/.netlify/gotrue/templates/confirm.html"},
		{"https://test.example.com/removedpath", "/.netlify/gotrue/templates", "confirm.html", "https://test.example.com/.netlify/gotrue/templates/confirm.html"},
		{"https://test.example.com/", "/trailingslash/", "confirm.html", "https://test.example.com/trailingslash/confirm.html"},
		{"https://test.example.com/", "/f/", "/extrafolder/confirm.html", "https://test.example.com/f/extrafolder/confirm.html"},
		{"https://test.example.com/", "f/", "/confirm.html", "https://test.example.com/f/confirm.html"},
		{"https://test.example.com/", "f", "/confirm.html", "https://test.example.com/f/confirm.html"},
		{"https://test.example.com", "f", "confirm.html", "https://test.example.com/f/confirm.html"},
	}

	for _, c := range cases {
		act, err := getSiteURL(c.SiteURL, c.Folder, c.Filename)
		assert.NoError(t, err, c.Expected)
		assert.Equal(t, c.Expected, act)
	}
}

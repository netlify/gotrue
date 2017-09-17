package api

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// List represents something
type Blacklist struct {
	domains   map[string]bool
	updatedAt *time.Time
}

// EmailAllowed Is email domain blacklisted
func (l *Blacklist) EmailBlacklisted(email string) bool {
	domain := strings.Split(email, "@")
	return len(domain) == 2 && l.domains[domain[1]]
}

// UpdateNeeded Does the blacklist need to be updated (in this case it's been over 24hours)
func (l *Blacklist) UpdateNeeded() bool {
	return l.updatedAt == nil || time.Now().After(l.updatedAt.AddDate(0, 0, 1))
}

// Update Update blacklist from provided blacklist file url
func (l *Blacklist) UpdateFromURL(blacklistUrl string) error {
	domains := make(map[string]bool)
	resp, err := http.Get(blacklistUrl)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	for _, domain := range strings.Split(string(body), "\n") {
		domains[domain] = true
	}
	l.Update(domains)
	return nil
}

// UpdateDomains Sets the domains that are part of the blacklist
func (l *Blacklist) Update(domains map[string]bool) {
	now := time.Now()
	l.domains = domains
	l.updatedAt = &now
}

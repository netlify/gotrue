package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	"gopkg.in/gomail.v2"

	"github.com/sirupsen/logrus"
)

// TemplateRetries is the amount of time MailMe will try to fetch a URL before giving up
const TemplateRetries = 3

// TemplateExpiration is the time period that the template will be cached for
const TemplateExpiration = 10 * time.Second

// Mailer lets MailMe send templated mails
type MailmeMailer struct {
	From    string
	Host    string
	Port    int
	User    string
	Pass    string
	BaseURL string
	FuncMap template.FuncMap
	cache   *TemplateCache
}

// Mail sends a templated mail. It will try to load the template from a URL, and
// otherwise fall back to the default
func (m *MailmeMailer) Mail(to, subjectTemplate, templateURL, defaultTemplate string, templateData map[string]interface{}) error {
	if m.FuncMap == nil {
		m.FuncMap = map[string]interface{}{}
	}
	if m.cache == nil {
		m.cache = &TemplateCache{templates: map[string]*MailTemplate{}, funcMap: m.FuncMap}
	}

	tmp, err := template.New("Subject").Funcs(template.FuncMap(m.FuncMap)).Parse(subjectTemplate)
	if err != nil {
		return err
	}

	subject := &bytes.Buffer{}
	err = tmp.Execute(subject, templateData)
	if err != nil {
		return err
	}
	body, err := m.MailBody(templateURL, defaultTemplate, templateData)
	if err != nil {
		return err
	}

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.From)
	mail.SetHeader("To", to)
	mail.SetHeader("Subject", subject.String())
	mail.SetBody("text/html", body)

	dial := gomail.NewPlainDialer(m.Host, m.Port, m.User, m.Pass)
	return dial.DialAndSend(mail)

}

type MailTemplate struct {
	tmp       *template.Template
	expiresAt time.Time
}

type TemplateCache struct {
	templates map[string]*MailTemplate
	mutex     sync.Mutex
	funcMap   template.FuncMap
}

func (t *TemplateCache) Get(url string) (*template.Template, error) {
	cached, ok := t.templates[url]
	if ok && (cached.expiresAt.Before(time.Now())) {
		return cached.tmp, nil
	}
	data, err := t.fetchTemplate(url, TemplateRetries)
	if err != nil {
		return nil, err
	}
	return t.Set(url, data, TemplateExpiration)
}

func (t *TemplateCache) Set(key, value string, expirationTime time.Duration) (*template.Template, error) {
	parsed, err := template.New(key).Funcs(t.funcMap).Parse(value)
	if err != nil {
		return nil, err
	}

	cached := &MailTemplate{
		tmp:       parsed,
		expiresAt: time.Now().Add(expirationTime),
	}
	t.mutex.Lock()
	t.templates[key] = cached
	t.mutex.Unlock()
	return parsed, nil
}

func (t *TemplateCache) fetchTemplate(url string, triesLeft int) (string, error) {
	client := &http.Client{}
	client.Transport = SafeRountripper(client.Transport, logrus.New())

	resp, err := client.Get(url)
	if err != nil && triesLeft > 0 {
		return t.fetchTemplate(url, triesLeft-1)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil && triesLeft > 0 {
			return t.fetchTemplate(url, triesLeft-1)
		}
		if err != nil {
			return "", err
		}
		return string(bodyBytes), err
	}
	if triesLeft > 0 {
		return t.fetchTemplate(url, triesLeft-1)
	}
	return "", errors.New("Unable to fetch mail template")
}

func (m *MailmeMailer) MailBody(url string, defaultTemplate string, data map[string]interface{}) (string, error) {
	if m.FuncMap == nil {
		m.FuncMap = map[string]interface{}{}
	}
	if m.cache == nil {
		m.cache = &TemplateCache{templates: map[string]*MailTemplate{}, funcMap: m.FuncMap}
	}

	var temp *template.Template
	var err error

	if url != "" {
		var absoluteURL string
		if strings.HasPrefix(url, "http") {
			absoluteURL = url
		} else {
			absoluteURL = m.BaseURL + url
		}
		temp, err = m.cache.Get(absoluteURL)
		if err != nil {
			log.Printf("Error loading template from %v: %v\n", url, err)
		}
	}

	if temp == nil {
		cached, ok := m.cache.templates[url]
		if ok {
			temp = cached.tmp
		} else {
			temp, err = m.cache.Set(url, defaultTemplate, 0)
			if err != nil {
				return "", err
			}
		}
	}

	buf := &bytes.Buffer{}
	err = temp.Execute(buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"100.64.0.0/10",  // RFC6598
		"172.16.0.0/12",  // RFC1918
		"192.0.0.0/24",   // RFC6890
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

type noLocalTransport struct {
	inner  http.RoundTripper
	errlog logrus.FieldLogger
}

func (no noLocalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())

	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if endpoint := isLocal(info); endpoint != "" {
				cancel()
				if no.errlog != nil {
					no.errlog.WithFields(logrus.Fields{
						"original_url":     req.URL.String(),
						"blocked_endpoint": endpoint,
					})
				}
			}
		},
	})

	req = req.WithContext(ctx)
	return no.inner.RoundTrip(req)
}

func isLocal(info httptrace.DNSDoneInfo) string {
	fmt.Printf("Got dns info: %v\n", info)
	for _, addr := range info.Addrs {
		fmt.Printf("Checking addr: %v\n", addr)
		if isPrivateIP(addr.IP) {
			return fmt.Sprintf("%v", addr.IP)
		}
	}
	return ""
}

func SafeRountripper(trans http.RoundTripper, log logrus.FieldLogger) http.RoundTripper {
	if trans == nil {
		trans = http.DefaultTransport
	}

	ret := &noLocalTransport{
		inner:  trans,
		errlog: log.WithField("transport", "local_blocker"),
	}

	return ret
}

func SafeHTTPClient(client *http.Client, log logrus.FieldLogger) *http.Client {
	client.Transport = SafeRountripper(client.Transport, log)

	return client
}

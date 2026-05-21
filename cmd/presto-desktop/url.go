package main

import (
	"log"
	"net/url"
	"regexp"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var prestoTemplateIDRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func (a *App) GetStartupURL() string {
	u := startupURL
	startupURL = ""
	logger.Debug("[url-scheme] GetStartupURL called", "url", u)
	return u
}

func (a *App) handlePrestoURL(rawURL string) {
	templateName, ok := parsePrestoTemplateURL(rawURL)
	if !ok {
		return
	}

	if !a.releaseCapabilities().OnlineTemplateStore {
		log.Printf("[url-scheme] template store disabled by release channel; rejecting template open: %s", templateName)
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "app:notification", map[string]string{
				"type":    "warning",
				"message": "离线便携版不支持从互联网打开模板商店内容。",
			})
		}
		return
	}

	log.Printf("[url-scheme] opening template: %s", templateName)
	wailsRuntime.EventsEmit(a.ctx, "url-scheme-open-template", templateName)
}

func parsePrestoTemplateURL(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[url-scheme] failed to parse URL: %s", rawURL)
		return "", false
	}

	if u.Scheme != "presto" {
		log.Printf("[url-scheme] unsupported scheme: %s", u.Scheme)
		return "", false
	}

	action := u.Host
	var templateName string
	switch action {
	case "open":
		query := u.Query()
		if u.Fragment != "" || !onlyQueryKeys(query, "resource", "id") {
			log.Printf("[url-scheme] unsupported open URL extras: %s", rawURL)
			return "", false
		}
		if query.Get("resource") != "template" {
			log.Printf("[url-scheme] unsupported open resource: %s", query.Get("resource"))
			return "", false
		}
		if u.Path != "" && u.Path != "/" {
			log.Printf("[url-scheme] unsupported open path: %s", u.Path)
			return "", false
		}
		templateName = query.Get("id")
	case "install":
		// Backward compatibility for existing website links.
		if u.RawQuery != "" || u.Fragment != "" {
			log.Printf("[url-scheme] unsupported legacy install URL extras: %s", rawURL)
			return "", false
		}
		templateName = strings.TrimPrefix(u.Path, "/")
	default:
		log.Printf("[url-scheme] unsupported action: %s", action)
		return "", false
	}

	if templateName == "" {
		log.Printf("[url-scheme] missing template name in URL: %s", rawURL)
		return "", false
	}
	if !prestoTemplateIDRe.MatchString(templateName) {
		log.Printf("[url-scheme] invalid template id: %s", templateName)
		return "", false
	}

	return templateName, true
}

func onlyQueryKeys(values url.Values, allowed ...string) bool {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range values {
		if _, ok := allowedSet[key]; !ok {
			return false
		}
	}
	return true
}

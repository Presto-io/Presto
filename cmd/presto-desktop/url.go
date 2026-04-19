package main

import (
	"log"
	"net/url"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetStartupURL() string {
	u := startupURL
	startupURL = ""
	logger.Debug("[url-scheme] GetStartupURL called", "url", u)
	return u
}

func (a *App) handlePrestoURL(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[url-scheme] failed to parse URL: %s", rawURL)
		return
	}

	action := u.Host
	if action != "install" {
		log.Printf("[url-scheme] unsupported action: %s", action)
		return
	}

	templateName := strings.TrimPrefix(u.Path, "/")
	if templateName == "" {
		log.Printf("[url-scheme] missing template name in URL: %s", rawURL)
		return
	}

	log.Printf("[url-scheme] opening template: %s", templateName)

	wailsRuntime.EventsEmit(a.ctx, "url-scheme-open-template", templateName)
}

package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
)

func main() {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		slog.Error("DOMAIN environment variable is required")
		os.Exit(1)
	}

	coreURL := os.Getenv("CORE_URL")
	if coreURL == "" {
		slog.Error("CORE_URL environment variable is required")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/callback", handleCallback(domain, coreURL))
	http.HandleFunc("/health", handleHealth)

	slog.Info("OAuth Proxy starting", "port", port, "domain", domain, "core_url", coreURL)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func handleCallback(domain string, coreURL string) http.HandlerFunc {
	// Compile regex once
	prRegex := regexp.MustCompile(`^(pr-\d+)-`)

	return func(w http.ResponseWriter, r *http.Request) {
		// Get OAuth parameters
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorParam := r.URL.Query().Get("error")

		// Handle OAuth errors from provider
		if errorParam != "" {
			errorDesc := r.URL.Query().Get("error_description")
			slog.Error("OAuth error from provider", "error", errorParam, "description", errorDesc)
			http.Error(w, fmt.Sprintf("OAuth Error: %s - %s", errorParam, errorDesc), http.StatusBadRequest)
			return
		}

		// Validate required parameters
		if code == "" || state == "" {
			slog.Warn("Missing required parameters", "has_code", code != "", "has_state", state != "")
			http.Error(w, "Missing required parameters: code and state", http.StatusBadRequest)
			return
		}

		// Check if this is a PR environment (state starts with "pr-{number}-")
		matches := prRegex.FindStringSubmatch(state)

		var targetURL string
		if len(matches) >= 2 {
			// PR environment - extract pr-{number} and redirect to PR backend
			prIdentifier := matches[1] // e.g., "pr-86"
			targetURL = fmt.Sprintf("https://%s-core.%s/login/callback?%s",
				prIdentifier, domain, r.URL.RawQuery)
			slog.Info("Redirecting OAuth callback to PR environment", "pr", prIdentifier, "target", targetURL)
		} else {
			// Production/staging - redirect to main backend
			targetURL = fmt.Sprintf("%s/login/callback?%s", coreURL, r.URL.RawQuery)
			slog.Info("Redirecting OAuth callback to production", "target", targetURL)
		}

		// Redirect to the backend with all query parameters
		http.Redirect(w, r, targetURL, http.StatusFound)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

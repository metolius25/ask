package provider

import (
	"crypto/tls"
	"net/http"
	"time"
)

// secureHTTPClient returns an HTTP client with explicit TLS verification
// and reasonable timeouts for API calls.
func secureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 120 * time.Second, // Overall request timeout
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false, // Explicitly verify certificates
			},
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
		},
	}
}

package client

import (
	"fmt"
	"net/http"
)

const apiVersion = "2025-11-05"

// Client is the Snyk REST API client for org and group memberships.
type Client struct {
	apiToken string
	baseURL  string
	http     *http.Client
}

// New creates a new Snyk API client.
func New(apiToken, baseURL string) (*Client, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("api_token is required")
	}
	return &Client{
		apiToken: apiToken,
		baseURL:  baseURL,
		http:     newHTTPClient(),
	}, nil
}

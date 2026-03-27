package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ListGroupSSOConnectionsResponse is the GET /groups/{group_id}/sso_connections response.
// See https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-sso_connections
type ListGroupSSOConnectionsResponse struct {
	Data []SSOConnectionItem `json:"data"`
}

// SSOConnectionItem is one SSO connection in the list (JSON:API resource).
type SSOConnectionItem struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ListGroupSSOConnectionUsersResponse is the GET /groups/{group_id}/sso_connections/{sso_id}/users response.
// See https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-sso_connections-sso_id-users
type ListGroupSSOConnectionUsersResponse struct {
	Data  []SSOConnectionUserItem `json:"data"`
	Links *ListResponseLinks      `json:"links,omitempty"`
}

// SSOConnectionUserItem is one user in the SSO connection users list (JSON:API resource).
type SSOConnectionUserItem struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ListGroupSSOConnections calls GET /rest/groups/{group_id}/sso_connections.
func (c *Client) ListGroupSSOConnections(ctx context.Context, groupID string) ([]SSOConnectionItem, error) {
	url := fmt.Sprintf("%s/rest/groups/%s/sso_connections?version=%s", c.baseURL, groupID, apiVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list group SSO connections: status %d: %s", resp.StatusCode, string(body))
	}

	var out ListGroupSSOConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Data, nil
}

// ListGroupSSOConnectionUsers calls GET /rest/groups/{group_id}/sso_connections/{sso_id}/users with paging (limit=100 per page).
func (c *Client) ListGroupSSOConnectionUsers(ctx context.Context, groupID, ssoID string) ([]SSOConnectionUserItem, error) {
	url := fmt.Sprintf("%s/rest/groups/%s/sso_connections/%s/users?version=%s&limit=100", c.baseURL, groupID, ssoID, apiVersion)
	var all []SSOConnectionUserItem
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}
		req.Header.Set("Authorization", "Token "+c.apiToken)
		req.Header.Set("Content-Type", "application/vnd.api+json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("list group SSO connection users: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListGroupSSOConnectionUsersResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		_ = resp.Body.Close()

		all = append(all, out.Data...)
		if out.Links == nil || out.Links.Next == "" {
			return all, nil
		}
		url = c.baseURL + "/rest" + out.Links.Next
	}
}

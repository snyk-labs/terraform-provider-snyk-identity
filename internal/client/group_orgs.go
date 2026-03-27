package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ListGroupOrgsResponse is the GET /groups/{group_id}/orgs response.
// See https://docs.snyk.io/snyk-api/reference/orgs#get-groups-group_id-orgs
type ListGroupOrgsResponse struct {
	Data  []OrgItem          `json:"data"`
	Links *ListResponseLinks `json:"links,omitempty"`
}

// OrgItem is one organization in the list (JSON:API resource).
type OrgItem struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ListGroupOrgs calls GET /rest/groups/{group_id}/orgs with paging (limit=100 per page).
func (c *Client) ListGroupOrgs(ctx context.Context, groupID string) ([]OrgItem, error) {
	url := fmt.Sprintf("%s/rest/groups/%s/orgs?version=%s&limit=100", c.baseURL, groupID, apiVersion)
	var all []OrgItem
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
			return nil, fmt.Errorf("list group orgs: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListGroupOrgsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		_ = resp.Body.Close()

		all = append(all, out.Data...)
		if out.Links == nil || out.Links.Next == "" {
			return all, nil
		}
		// the response links are already prefixed with "/rest" so don't add it again
		url = c.baseURL + out.Links.Next
	}
}

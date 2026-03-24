package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RoleItem is one role in the v1 GET /group/{group_id}/roles response (plain JSON array).
// See https://docs.snyk.io/snyk-api/reference/groups-v1#get-group-groupid-roles
type RoleItem struct {
	Name       string `json:"name"`
	CustomRole bool   `json:"customRole"`
	PublicID   string `json:"publicId"`
}

// ListGroupRolesV1 calls GET /v1/group/{group_id}/roles (v1 API; no paging, single response).
// this only gets all the org roles, not the group roles
func (c *Client) ListGroupRolesV1(groupID string) ([]RoleItem, error) {
	url := fmt.Sprintf("%s/v1/group/%s/roles", c.baseURL, groupID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list group roles: status %d: %s", resp.StatusCode, string(body))
	}

	var out []RoleItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out, nil
}

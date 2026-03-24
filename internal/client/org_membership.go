package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateOrgMembershipRequest is the JSON:API request body for POST /orgs/{org_id}/memberships.
type CreateOrgMembershipRequest struct {
	Data CreateOrgMembershipData `json:"data"`
}

// CreateOrgMembershipData is the data object for creating an org membership.
type CreateOrgMembershipData struct {
	Type          string                  `json:"type"`
	Relationships CreateOrgMembershipRels `json:"relationships"`
}

// CreateOrgMembershipRels holds org, user and role relationships.
type CreateOrgMembershipRels struct {
	Org  RelationshipData `json:"org"`
	User RelationshipData `json:"user"`
	Role RelationshipData `json:"role"`
}

// CreateOrgMembershipResponse is the 201 response body. The API returns the
// resource at the top level (not under "data"): id, type, attributes, relationships.
type CreateOrgMembershipResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ListOrgMembershipsResponse is the GET /orgs/{org_id}/memberships response.
type ListOrgMembershipsResponse struct {
	Data  []ListOrgMembershipItem `json:"data"`
	Links *ListResponseLinks      `json:"links,omitempty"`
}

// ListOrgMembershipItem is one membership in the list.
type ListOrgMembershipItem struct {
	ID            string                         `json:"id"`
	Type          string                         `json:"type"`
	Relationships ListOrgMembershipRelationships `json:"relationships"`
}

// UpdateOrgMembershipRequest is the JSON:API request body for PATCH /orgs/{org_id}/memberships/{membership_id}.
// See https://apidocs.snyk.io/?version=2025-11-05#patch-/orgs/-org_id-/memberships/-membership_id-
type UpdateOrgMembershipRequest struct {
	Data UpdateOrgMembershipData `json:"data"`
}

// UpdateOrgMembershipData is the data object for updating an org membership (e.g. role).
type UpdateOrgMembershipData struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Relationships UpdateOrgMembershipRels `json:"relationships"`
}

// UpdateOrgMembershipRels holds only the role relationship for PATCH (role is the updatable field).
type UpdateOrgMembershipRels struct {
	Role RelationshipData `json:"role"`
}

// CreateOrgMembership calls POST /rest/orgs/{org_id}/memberships.
// See https://docs.snyk.io/snyk-api/reference/orgs#post-orgs-org_id-memberships
func (c *Client) CreateOrgMembership(orgID, userID, roleID string) (membershipID string, err error) {
	url := fmt.Sprintf("%s/rest/orgs/%s/memberships?version=%s", c.baseURL, orgID, apiVersion)

	body := CreateOrgMembershipRequest{
		Data: CreateOrgMembershipData{
			Type: "org_membership",
			Relationships: CreateOrgMembershipRels{
				Org:  RelationshipData{Data: ResourceRef{Type: "org", ID: orgID}},
				User: RelationshipData{Data: ResourceRef{Type: "user", ID: userID}},
				Role: RelationshipData{Data: ResourceRef{Type: "org_role", ID: roleID}},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create org membership: status %d: %s", resp.StatusCode, string(respBody))
	}

	var out CreateOrgMembershipResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return out.ID, nil
}

// ListOrgMemberships calls GET /rest/orgs/{org_id}/memberships with paging (limit=100 per page) and returns all memberships.
// See https://docs.snyk.io/snyk-api/reference/orgs#get-orgs-org_id-memberships
func (c *Client) ListOrgMemberships(orgID string) ([]ListOrgMembershipItem, error) {
	reqURL := fmt.Sprintf("%s/rest/orgs/%s/memberships?version=%s&limit=100", c.baseURL, orgID, apiVersion)
	var all []ListOrgMembershipItem
	for {
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
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
			return nil, fmt.Errorf("list org memberships: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListOrgMembershipsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		_ = resp.Body.Close()

		all = append(all, out.Data...)
		if out.Links == nil || out.Links.Next == "" {
			return all, nil
		}
		reqURL = c.baseURL + "/rest" + out.Links.Next
	}
}

// GetOrgMembershipByID lists memberships with paging and returns the one with the given ID, or nil.
func (c *Client) GetOrgMembershipByID(orgID, membershipID string) (*ListOrgMembershipItem, error) {
	url := fmt.Sprintf("%s/rest/orgs/%s/memberships?version=%s&limit=100", c.baseURL, orgID, apiVersion)
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
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
			return nil, fmt.Errorf("list org memberships: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListOrgMembershipsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		_ = resp.Body.Close()

		for i := range out.Data {
			if out.Data[i].ID == membershipID {
				return &out.Data[i], nil
			}
		}
		if out.Links == nil || out.Links.Next == "" {
			return nil, nil // not found on any page
		}
		url = c.baseURL + "/rest" + out.Links.Next
	}
}

// DeleteOrgMembership calls DELETE /rest/orgs/{org_id}/memberships/{membership_id}.
func (c *Client) DeleteOrgMembership(orgID, membershipID string) error {
	url := fmt.Sprintf("%s/rest/orgs/%s/memberships/%s?version=%s", c.baseURL, orgID, membershipID, apiVersion)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete org membership: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// UpdateOrgMembership calls PATCH /rest/orgs/{org_id}/memberships/{membership_id} to update the membership (e.g. role_id).
func (c *Client) UpdateOrgMembership(orgID, membershipID, roleID string) error {
	url := fmt.Sprintf("%s/rest/orgs/%s/memberships/%s?version=%s", c.baseURL, orgID, membershipID, apiVersion)

	body := UpdateOrgMembershipRequest{
		Data: UpdateOrgMembershipData{
			Type: "org_membership",
			ID:   membershipID,
			Relationships: UpdateOrgMembershipRels{
				Role: RelationshipData{Data: ResourceRef{Type: "org_role", ID: roleID}},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("update org membership: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

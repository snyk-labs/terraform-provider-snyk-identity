package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CreateGroupMembershipRequest is the JSON:API request body for POST /groups/{group_id}/memberships.
type CreateGroupMembershipRequest struct {
	Data CreateGroupMembershipData `json:"data"`
}

// CreateGroupMembershipData is the data object for creating a group membership.
type CreateGroupMembershipData struct {
	Type          string                    `json:"type"`
	Relationships CreateGroupMembershipRels `json:"relationships"` // same shape: user + role
}

// CreateGroupMembershipRels holds group, user and role relationships.
type CreateGroupMembershipRels struct {
	Group RelationshipData `json:"group"`
	User  RelationshipData `json:"user"`
	Role  RelationshipData `json:"role"`
}

// CreateGroupMembershipResponse is the 201 response body.
type CreateGroupMembershipResponse struct {
	Data GroupMembershipResource `json:"data"`
}

// GroupMembershipResource is the membership resource in the response.
type GroupMembershipResource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// GetGroupResponse is the GET /groups/{group_id} response (JSON:API).
// See https://docs.snyk.io/snyk-api/reference/group#get-groups-group_id
type GetGroupResponse struct {
	Data GetGroupData `json:"data"`
}

// GetGroupData is the group resource (id, type, attributes).
type GetGroupData struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ListGroupMembershipsResponse is the GET /groups/{group_id}/memberships response.
type ListGroupMembershipsResponse struct {
	Data  []ListGroupMembershipItem `json:"data"`
	Links *ListResponseLinks        `json:"links,omitempty"`
}

// ListGroupMembershipItem is one membership in the list.
type ListGroupMembershipItem struct {
	ID            string                         `json:"id"`
	Type          string                         `json:"type"`
	Relationships ListOrgMembershipRelationships `json:"relationships"`
}

// UpdateGroupMembershipRequest is the JSON:API request body for PATCH /groups/{group_id}/memberships/{membership_id}.
// See https://apidocs.snyk.io/?version=2025-11-05#patch-/groups/-group_id-/memberships/-membership_id-
type UpdateGroupMembershipRequest struct {
	Data UpdateGroupMembershipData `json:"data"`
}

// UpdateGroupMembershipData is the data object for updating a group membership (e.g. role).
type UpdateGroupMembershipData struct {
	Type          string                    `json:"type"`
	ID            string                    `json:"id"`
	Relationships UpdateGroupMembershipRels `json:"relationships"`
}

// UpdateGroupMembershipRels holds only the role relationship for PATCH (role is the updatable field).
type UpdateGroupMembershipRels struct {
	Role RelationshipData `json:"role"`
}

// GetGroup calls GET /rest/groups/{group_id} and returns the group details.
func (c *Client) GetGroup(groupID string) (*GetGroupData, error) {
	url := fmt.Sprintf("%s/rest/groups/%s?version=%s", c.baseURL, groupID, apiVersion)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get group: status %d: %s", resp.StatusCode, string(body))
	}

	var out GetGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out.Data, nil
}

// CreateGroupMembership calls POST /rest/groups/{group_id}/memberships.
// See https://docs.snyk.io/snyk-api/reference/groups#post-groups-group_id-memberships
// If the API returns 409 Conflict (e.g. a default membership already exists), the client updates that
// membership instead of creating a new one.
func (c *Client) CreateGroupMembership(groupID, userID, roleID string) (membershipID string, err error) {
	url := fmt.Sprintf("%s/rest/groups/%s/memberships?version=%s", c.baseURL, groupID, apiVersion)

	body := CreateGroupMembershipRequest{
		Data: CreateGroupMembershipData{
			Type: "group_membership",
			Relationships: CreateGroupMembershipRels{
				Group: RelationshipData{Data: ResourceRef{Type: "group", ID: groupID}},
				User:  RelationshipData{Data: ResourceRef{Type: "user", ID: userID}},
				Role:  RelationshipData{Data: ResourceRef{Type: "group_role", ID: roleID}},
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
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusConflict {
		existingID, err := c.getGroupMembershipIDOfUser(groupID, userID)
		if err != nil {
			return "", fmt.Errorf("get group membership of user error: %w", err)
		}
		if existingID == "" {
			return "", fmt.Errorf("create group membership: status 409: could not resolve existing membership: %s", string(respBody))
		}
		if err := c.UpdateGroupMembership(groupID, existingID, roleID); err != nil {
			return "", fmt.Errorf("update group membership error: %w", err)
		}
		return existingID, nil
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create group membership: status %d: %s", resp.StatusCode, string(respBody))
	}

	var out CreateGroupMembershipResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return out.Data.ID, nil
}

// ListGroupMemberships calls GET /rest/groups/{group_id}/memberships with paging (limit=100 per page) and returns all memberships.
// See https://docs.snyk.io/snyk-api/reference/groups#get-groups-group_id-memberships
func (c *Client) ListGroupMemberships(groupID string) ([]ListGroupMembershipItem, error) {
	reqURL := fmt.Sprintf("%s/rest/groups/%s/memberships?version=%s&limit=100", c.baseURL, groupID, apiVersion)
	var all []ListGroupMembershipItem
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
			resp.Body.Close()
			return nil, fmt.Errorf("list group memberships: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListGroupMembershipsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		all = append(all, out.Data...)
		if out.Links == nil || out.Links.Next == "" {
			return all, nil
		}
		reqURL = c.baseURL + "/rest" + out.Links.Next
	}
}

// GetGroupMembershipByID lists group memberships with paging and returns the one with the given ID, or nil.
func (c *Client) GetGroupMembershipByID(groupID, membershipID string) (*ListGroupMembershipItem, error) {
	url := fmt.Sprintf("%s/rest/groups/%s/memberships?version=%s&limit=100", c.baseURL, groupID, apiVersion)
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
			resp.Body.Close()
			return nil, fmt.Errorf("list group memberships: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListGroupMembershipsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

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

// getGroupMembershipIDOfUser returns the group_membership id for userID in groupID, or "" if not listed.
// Uses GET /groups/{group_id}/memberships with the user_id query parameter.
func (c *Client) getGroupMembershipIDOfUser(groupID, userID string) (string, error) {
	q := url.Values{}
	q.Set("version", apiVersion)
	q.Set("user_id", userID)
	q.Set("limit", "10")
	reqURL := fmt.Sprintf("%s/rest/groups/%s/memberships?%s", c.baseURL, groupID, q.Encode())
	for {
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return "", fmt.Errorf("new request: %w", err)
		}
		req.Header.Set("Authorization", "Token "+c.apiToken)
		req.Header.Set("Content-Type", "application/vnd.api+json")

		resp, err := c.http.Do(req)
		if err != nil {
			return "", fmt.Errorf("request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return "", fmt.Errorf("list group memberships: status %d: %s", resp.StatusCode, string(body))
		}

		var out ListGroupMembershipsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		if len(out.Data) > 0 {
			return out.Data[0].ID, nil
		}
		if out.Links == nil || out.Links.Next == "" {
			return "", nil
		}
		reqURL = c.baseURL + "/rest" + out.Links.Next
	}
}

// UpdateGroupMembership calls PATCH /rest/groups/{group_id}/memberships/{membership_id} to update the membership (e.g. role_id).
func (c *Client) UpdateGroupMembership(groupID, membershipID, roleID string) error {
	url := fmt.Sprintf("%s/rest/groups/%s/memberships/%s?version=%s", c.baseURL, groupID, membershipID, apiVersion)

	body := UpdateGroupMembershipRequest{
		Data: UpdateGroupMembershipData{
			Type: "group_membership",
			ID:   membershipID,
			Relationships: UpdateGroupMembershipRels{
				Role: RelationshipData{Data: ResourceRef{Type: "group_role", ID: roleID}},
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
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("update group membership: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteGroupMembership calls DELETE /rest/groups/{group_id}/memberships/{membership_id}.
// cascade: if true, also deletes child org memberships of the group membership.
func (c *Client) DeleteGroupMembership(groupID, membershipID string, cascade bool) error {
	url := fmt.Sprintf("%s/rest/groups/%s/memberships/%s?version=%s", c.baseURL, groupID, membershipID, apiVersion)
	if cascade {
		url += "&cascade=true"
	}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete group membership: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

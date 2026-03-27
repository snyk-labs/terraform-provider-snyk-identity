package resources

import (
	"testing"

	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

func TestParseOrgMembershipImportID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id      string
		wantOrg string
		wantMem string
		wantOK  bool
	}{
		{"org-1/mem-2", "org-1", "mem-2", true},
		{"a/b/c", "a", "b/c", true},
		{"", "", "", false},
		{"no-slash", "", "", false},
		{"/only", "", "", false},
	}
	for _, tt := range tests {
		o, m, ok := parseOrgMembershipImportID(tt.id)
		if ok != tt.wantOK || o != tt.wantOrg || m != tt.wantMem {
			t.Errorf("parseOrgMembershipImportID(%q) = (%q,%q,%v), want (%q,%q,%v)", tt.id, o, m, ok, tt.wantOrg, tt.wantMem, tt.wantOK)
		}
	}
}

func TestParseGroupMembershipImportID(t *testing.T) {
	t.Parallel()
	g, m, ok := parseGroupMembershipImportID("g1/m1")
	if !ok || g != "g1" || m != "m1" {
		t.Errorf("got %q %q %v", g, m, ok)
	}
}

func TestExtractUserAndRoleFromMembership(t *testing.T) {
	t.Parallel()
	if u, r := extractUserAndRoleFromMembership(nil); u != "" || r != "" {
		t.Errorf("nil: %q %q", u, r)
	}
	m := &client.ListOrgMembershipItem{
		Relationships: client.ListOrgMembershipRelationships{
			User: client.RelationshipData{Data: client.ResourceRef{ID: "uu"}},
			Role: client.RelationshipData{Data: client.ResourceRef{ID: "rr"}},
		},
	}
	if u, r := extractUserAndRoleFromMembership(m); u != "uu" || r != "rr" {
		t.Errorf("got %q %q", u, r)
	}
}

func TestExtractUserAndRoleFromGroupMembership(t *testing.T) {
	t.Parallel()
	if u, r := extractUserAndRoleFromGroupMembership(nil); u != "" || r != "" {
		t.Errorf("nil: %q %q", u, r)
	}
	m := &client.ListGroupMembershipItem{
		Relationships: client.ListOrgMembershipRelationships{
			User: client.RelationshipData{Data: client.ResourceRef{ID: "uu"}},
			Role: client.RelationshipData{Data: client.ResourceRef{ID: "rr"}},
		},
	}
	if u, r := extractUserAndRoleFromGroupMembership(m); u != "uu" || r != "rr" {
		t.Errorf("got %q %q", u, r)
	}
}

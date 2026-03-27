package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/snyk-labs/terraform-provider-snyk-identity/internal/client"
)

func TestNew(t *testing.T) {
	t.Parallel()
	_, err := client.New("", "https://api.snyk.io")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	c, err := client.New("tok", "https://api.snyk.io")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}

func TestCreateOrgMembership(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Token ") {
			t.Error("missing Token auth")
		}
		if !strings.Contains(r.URL.Path, "/rest/orgs/org-1/memberships") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"mem-1","type":"org_membership"}`))
	}))
	defer ts.Close()

	c, err := client.New("tok", ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.CreateOrgMembership(t.Context(), "org-1", "user-1", "role-1")
	if err != nil {
		t.Fatal(err)
	}
	if id != "mem-1" {
		t.Errorf("id %q", id)
	}
}

func TestCreateOrgMembership_errorStatus(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "bad")
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	_, err := c.CreateOrgMembership(t.Context(), "o", "u", "r")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetOrgMembershipByID_firstPage(t *testing.T) {
	t.Parallel()
	body := `{"data":[{"id":"m1","type":"org_membership","relationships":{"user":{"data":{"type":"user","id":"u1"}},"role":{"data":{"type":"org_role","id":"r1"}}}}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s", r.Method)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	m, err := c.GetOrgMembershipByID(t.Context(), "org-1", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.ID != "m1" {
		t.Fatalf("got %+v", m)
	}
}

func TestGetOrgMembershipByID_secondPage(t *testing.T) {
	t.Parallel()
	page1 := `{"data":[{"id":"other","type":"org_membership","relationships":{"user":{"data":{"type":"user","id":"u"}},"role":{"data":{"type":"org_role","id":"r"}}}}],"links":{"next":"/orgs/org-1/memberships?cursor=2"}}`
	page2 := `{"data":[{"id":"m2","type":"org_membership","relationships":{"user":{"data":{"type":"user","id":"u2"}},"role":{"data":{"type":"org_role","id":"r2"}}}}]}`
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(page1))
			return
		}
		_, _ = w.Write([]byte(page2))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	m, err := c.GetOrgMembershipByID(t.Context(), "org-1", "m2")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.ID != "m2" {
		t.Fatalf("got %+v", m)
	}
	if calls != 2 {
		t.Errorf("calls %d", calls)
	}
}

func TestGetOrgMembershipByID_notFound(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	m, err := c.GetOrgMembershipByID(t.Context(), "org-1", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil")
	}
}

func TestListOrgMemberships_paging(t *testing.T) {
	t.Parallel()
	item := `{"id":"m1","type":"org_membership","relationships":{"user":{"data":{"type":"user","id":"u1"}},"role":{"data":{"type":"org_role","id":"r1"}}}}`
	p1 := `{"data":[` + item + `],"links":{"next":"/orgs/o1/memberships?cursor=2"}}`
	p2 := `{"data":[{"id":"m2","type":"org_membership","relationships":{"user":{"data":{"type":"user","id":"u2"}},"role":{"data":{"type":"org_role","id":"r2"}}}}]}`
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write([]byte(p1))
			return
		}
		_, _ = w.Write([]byte(p2))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	all, err := c.ListOrgMemberships(t.Context(), "o1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "m1" || all[1].ID != "m2" {
		t.Fatalf("got %+v", all)
	}
	if n != 2 {
		t.Errorf("requests %d", n)
	}
}

func TestDeleteOrgMembership(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	if err := c.DeleteOrgMembership(t.Context(), "o", "m"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOrgMembership(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	if err := c.UpdateOrgMembership(t.Context(), "o", "m", "r"); err != nil {
		t.Fatal(err)
	}
}

func TestGetGroup(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":"g1","type":"group","attributes":{"name":"n"}}}`))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	g, err := c.GetGroup(t.Context(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	if g.ID != "g1" {
		t.Errorf("id %q", g.ID)
	}
}

func TestCreateGroupMembership_created(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"gm1","type":"group_membership"}}`))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	id, err := c.CreateGroupMembership(t.Context(), "g1", "u1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if id != "gm1" {
		t.Errorf("id %q", id)
	}
}

func TestCreateGroupMembership_conflictThenUpdate(t *testing.T) {
	t.Parallel()
	var postCount, getCount, patchCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/memberships"):
			postCount++
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, "exists")
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/memberships") && r.URL.Query().Get("user_id") != "":
			getCount++
			body := `{"data":[{"id":"existing","type":"group_membership","relationships":{"user":{"data":{"type":"user","id":"u1"}},"role":{"data":{"type":"group_role","id":"r0"}}}}]}`
			_, _ = w.Write([]byte(body))
		case r.Method == http.MethodPatch:
			patchCount++
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	id, err := c.CreateGroupMembership(t.Context(), "g1", "u1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if id != "existing" {
		t.Errorf("id %q", id)
	}
	if postCount != 1 || getCount != 1 || patchCount != 1 {
		t.Errorf("post %d get %d patch %d", postCount, getCount, patchCount)
	}
}

func TestGetGroupMembershipByID(t *testing.T) {
	t.Parallel()
	body := `{"data":[{"id":"gm","type":"group_membership","relationships":{"user":{"data":{"type":"user","id":"u"}},"role":{"data":{"type":"group_role","id":"r"}}}}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	m, err := c.GetGroupMembershipByID(t.Context(), "g", "gm")
	if err != nil || m == nil || m.ID != "gm" {
		t.Fatalf("err=%v m=%+v", err, m)
	}
}

func TestListGroupMemberships_paging(t *testing.T) {
	t.Parallel()
	item := `{"id":"m1","type":"group_membership","relationships":{"user":{"data":{"type":"user","id":"u1"}},"role":{"data":{"type":"group_role","id":"r1"}}}}`
	p1 := `{"data":[` + item + `],"links":{"next":"/groups/g/memberships?cursor=2"}}`
	p2 := `{"data":[{"id":"m2","type":"group_membership","relationships":{"user":{"data":{"type":"user","id":"u2"}},"role":{"data":{"type":"group_role","id":"r2"}}}}]}`
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write([]byte(p1))
			return
		}
		_, _ = w.Write([]byte(p2))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	all, err := c.ListGroupMemberships(t.Context(), "g")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "m1" || all[1].ID != "m2" {
		t.Fatalf("got %+v", all)
	}
	if n != 2 {
		t.Errorf("requests %d", n)
	}
}

func TestDeleteGroupMembership_cascadeQuery(t *testing.T) {
	t.Parallel()
	var sawCascade bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawCascade = r.URL.Query().Get("cascade") == "true"
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	if err := c.DeleteGroupMembership(t.Context(), "g", "m", true); err != nil {
		t.Fatal(err)
	}
	if !sawCascade {
		t.Error("expected cascade=true")
	}
}

func TestListGroupSSOConnections(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"s1","type":"sso_connection"}]}`))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	list, err := c.ListGroupSSOConnections(t.Context(), "g")
	if err != nil || len(list) != 1 || list[0].ID != "s1" {
		t.Fatalf("err=%v list=%+v", err, list)
	}
}

func TestListGroupSSOConnectionUsers_paging(t *testing.T) {
	t.Parallel()
	p1 := `{"data":[{"id":"u1","type":"user"}],"links":{"next":"/groups/g/sso_connections/s/users?page=2"}}`
	p2 := `{"data":[{"id":"u2","type":"user"}]}`
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write([]byte(p1))
			return
		}
		_, _ = w.Write([]byte(p2))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	all, err := c.ListGroupSSOConnectionUsers(t.Context(), "g", "s")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != "u1" || all[1].ID != "u2" {
		t.Fatalf("got %+v", all)
	}
	if n != 2 {
		t.Errorf("requests %d", n)
	}
}

func TestListGroupOrgs_paging(t *testing.T) {
	t.Parallel()
	p1 := `{"data":[{"id":"o1","type":"org"}],"links":{"next":"/rest/groups/g/orgs?cursor=2"}}`
	p2 := `{"data":[{"id":"o2","type":"org"}]}`
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write([]byte(p1))
			return
		}
		_, _ = w.Write([]byte(p2))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	all, err := c.ListGroupOrgs(t.Context(), "g")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("len %d", len(all))
	}
}

func TestListGroupRolesV1(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v1/group/g/roles") {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"admin","customRole":false,"publicId":"rid"}]`))
	}))
	defer ts.Close()

	c, _ := client.New("tok", ts.URL)
	roles, err := c.ListGroupRolesV1(t.Context(), "g")
	if err != nil || len(roles) != 1 || roles[0].PublicID != "rid" {
		t.Fatalf("err=%v %+v", err, roles)
	}
}

func TestJSONTypes_roundTrip(t *testing.T) {
	t.Parallel()
	req := client.CreateOrgMembershipRequest{
		Data: client.CreateOrgMembershipData{
			Type: "org_membership",
			Relationships: client.CreateOrgMembershipRels{
				Org:  client.RelationshipData{Data: client.ResourceRef{Type: "org", ID: "o"}},
				User: client.RelationshipData{Data: client.ResourceRef{Type: "user", ID: "u"}},
				Role: client.RelationshipData{Data: client.ResourceRef{Type: "org_role", ID: "r"}},
			},
		},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back client.CreateOrgMembershipRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Data.Type != "org_membership" {
		t.Fatal("round-trip failed")
	}
}

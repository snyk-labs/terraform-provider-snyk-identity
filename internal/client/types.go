package client

// RelationshipData is JSON:API relationship reference.
type RelationshipData struct {
	Data ResourceRef `json:"data"`
}

// ResourceRef is type + id.
type ResourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ListResponseLinks holds optional "next" URL path for paging.
type ListResponseLinks struct {
	Next string `json:"next,omitempty"`
}

// ListOrgMembershipRelationships holds user and role refs in list responses
// (used by both org and group membership list items).
type ListOrgMembershipRelationships struct {
	User RelationshipData `json:"user"`
	Role RelationshipData `json:"role"`
}

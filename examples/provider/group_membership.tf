# Example: snyk_group_membership with group-level data sources (group metadata, SSO).
# Group role UUID usually comes from output role_public_ids (main.tf) or your tenant configuration.

variable "group_role_id" {
  type        = string
  description = "Group role UUID (often from output role_public_ids; group roles vs org roles depend on your tenant)"
}

variable "cascade_delete" {
  type        = bool
  default     = false
  description = "When true, deleting this membership also removes the user's org memberships in the group"
}

data "snyk_group" "this" {
  group_id = var.group_id
}

data "snyk_group_memberships" "all_group_memberships" {
  group_id = var.group_id
}

data "snyk_sso_connections" "group_sso" {
  group_id = var.group_id
}

# List SSO connection users of Self-Serve Single Sign-On (SSO) connection at Group level
data "snyk_sso_connection_users" "sso_users" {
  count    = vdata.snyk_sso_connections.group_sso.connections[0].id != "" ? 1 : 0
  group_id = var.group_id
  sso_id   = data.snyk_sso_connections.group_sso.connections[0].id
}

resource "snyk_group_membership" "member" {
  group_id       = var.group_id
  user_id        = var.user_id
  role_id        = var.group_role_id
  cascade_delete = var.cascade_delete
}

output "group_membership_id" {
  value       = snyk_group_membership.member.id
  description = "Created group membership UUID"
}

output "group_name" {
  value       = data.snyk_group.this.name
  description = "Group display name from data.snyk_group"
}

output "group_slug" {
  value       = data.snyk_group.this.slug
  description = "Group URL slug from data.snyk_group"
}

output "all_group_memberships" {
  value       = data.snyk_group_memberships.all_group_memberships.memberships
  description = "All group memberships (id, type, user_id, role_id) from data.snyk_group_memberships"
}

output "sso_connections" {
  value       = data.snyk_sso_connections.group_sso.connections
  description = "SSO connections for the group (data.snyk_sso_connections)"
}

# Get Self-Serve Single Sign-On (SSO) connection ID at Group level
output "sso_connection_id" {
  value       = length(data.snyk_sso_connections.group_sso.connections) > 0 ? data.snyk_sso_connections.group_sso.connections[0].id : null
  description = "Snyk Self-Serve Single Sign-On (SSO) connection ID"
}

output "sso_connection_users" {
  value       = data.snyk_sso_connection_users.sso_users.users
  description = "Snyk Self-Serve Single Sign-On (SSO) connection users"
}

output "all_sso_connection_users_id" {
  description = "Distinct Self-Serve Single Sign-On (SSO) connection users IDs."
  value       = distinct([for u in data.snyk_sso_connection_users.sso_users.users : u.id])
}

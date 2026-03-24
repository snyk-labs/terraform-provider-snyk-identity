terraform {
  required_providers {
    snyk = {
      source  = "snyk-labs/snyk-identity"
      version = "0.1.0"
    }
  }
}

provider "snyk" {
  api_token = var.api_token
}

# --- Shared inputs for both membership examples ---

variable "api_token" {
  type      = string
  sensitive = true
}

variable "group_id" {
  type        = string
  description = "Snyk group UUID (scopes shared data sources; group membership is created in this group)"
}

variable "user_id" {
  type        = string
  description = "User UUID to add as org and/or group member"
}

# --- Shared data: discover orgs and roles in the group (use outputs to pick IDs for resources below) ---

data "snyk_orgs" "in_group" {
  group_id = var.group_id
}

data "snyk_roles" "in_group" {
  group_id = var.group_id
}

output "orgs_in_group" {
  value       = data.snyk_orgs.in_group.orgs
  description = "Organizations in the group; use an org id for snyk_org_membership (see org_membership.tf)"
}

output "roles_in_group" {
  value       = data.snyk_roles.in_group.roles
  description = "Roles from the group roles API; use public_id for org_role_id / group_role_id"
}

output "org_ids_in_group" {
  value       = [for o in data.snyk_orgs.in_group.orgs : o.id]
  description = "Organization UUIDs under the group"
}

output "role_public_ids" {
  value       = [for r in data.snyk_roles.in_group.roles : r.public_id]
  description = "Role UUIDs (public_id) available when assigning memberships"
}

# Example: snyk_org_membership driven by group-scoped data (see main.tf: data.snyk_orgs, data.snyk_roles).
# Set org_id and org_role_id from terraform output after refresh (e.g. org_ids_in_group[0], role_public_ids[0]),
# or pass explicit UUIDs via .tfvars.

variable "org_id" {
  type        = string
  description = "Target organization UUID (typically one of the values from output org_ids_in_group)"
}

variable "org_role_id" {
  type        = string
  description = "Org role UUID (typically one of the values from output role_public_ids)"
}

data "snyk_org_memberships" "all" {
  org_id = var.org_id
}

resource "snyk_org_membership" "member" {
  org_id  = var.org_id
  user_id = var.user_id
  role_id = var.org_role_id
}

output "org_membership_id" {
  value       = snyk_org_membership.member.id
  description = "Created org membership UUID"
}

output "org_memberships" {
  value       = data.snyk_org_memberships.all.memberships
  description = "All org memberships (id, type, user_id, role_id) from data.snyk_org_memberships"
}

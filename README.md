# Snyk Identity Terraform Provider

Terraform provider for managing Snyk organization and group memberships, plus read-only data sources for groups, orgs, roles, and SSO connection details. It uses the Snyk REST API (default version `2025-11-05` where applicable).

## Features

This provider is primarily used to automate Snyk users access provisioning after [Configure Self-Serve Single Sign-On (SSO)](https://docs.snyk.io/snyk-platform-administration/single-sign-on-sso-for-authentication-to-snyk/configure-self-serve-single-sign-on-sso) through following functions.

- *Group Membership*: Create and manage Snyk user group membership
- *Org Membership*: Create and manage Snyk user org membership
- *Group*: Get Group details
- *Orgs*: List organizations of a Group
- *Roles*: List Snyk organization Roles of a Group
- *Connections*: List SSO connection and SSO connection users

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24+ (for building from source)
- Snyk API Token with Group Admin access

### Build from Source

```bash
# Clone the repository
git clone https://github.com/snyk-labs/terraform-provider-snyk-identity.git
cd terraform-provider-snyk-identity

# Build and install locally
make build
```
Place the binary in your [Terraform plugin directory](https://developer.hashicorp.com/terraform/plugin/how-terraform-works#plugin-locations) or use a [development override](https://developer.hashicorp.com/terraform/plugin/development#development-overrides).

## Installation

### From Terraform Registry

```hcl
terraform {
  required_providers {
    snyk = {
      source  = "snyk-labs/snyk-identity"
      version = "~> 0.1"
    }
  }
}
```

## Provider configuration

Declare the provider and pass a Snyk API token. Optional `api_endpoint` sets the API host (for example `api.snyk.io`); if you omit a scheme, `https://` is prepended. When unset, the client uses `https://api.snyk.io`.

```hcl
provider "snyk" {
  api_token = var.api_token
  # optional Snyk API endpoint variable
  # api_endpoint = "api.snyk.io"
}
```

Grant the token the scopes your configuration needs (for example `org.membership.add` / `group.membership.add` for the membership resources).

## Resources

| Resource | Description |
|----------|-------------|
| `snyk_org_membership` | Creates and manages an organization membership (user + org role). |
| `snyk_group_membership` | Creates and manages a group membership (user + group role), with optional cascade delete on destroy. |

## Data sources

| Data source | Description |
|-------------|-------------|
| `snyk_group` | Reads a group by ID (attributes such as name when returned by the API). |
| `snyk_orgs` | Lists organizations in a group. |
| `snyk_org_memberships` | Lists all memberships of an organization (user and org role per membership). |
| `snyk_roles` | Lists organization roles for a group. |
| `snyk_sso_connections` | Lists SSO connections configured for a group. |
| `snyk_sso_connection_users` | Lists users for a given SSO connection in a group. |
| `snyk_group_memberships` | Lists all memberships of a group (user and role per membership). |

## Import

Imports use the resource address and a single import ID string.

### `snyk_org_membership`

Use **`org_id`/`membership_id`** (two UUIDs separated by one slash):

```bash
terraform import snyk_org_membership.example "<org-uuid>/<membership-uuid>"
```

Example:

```bash
terraform import snyk_org_membership.member "b667f176-df52-4b0a-9954-117af6b05ab7/550e8400-e29b-41d4-a716-446655440000"
```

The provider loads the membership via the org memberships API and sets `user_id` and `role_id` from the response.

### `snyk_group_membership`

Use **`group_id`/`membership_id`** (two UUIDs separated by one slash):

```bash
terraform import snyk_group_membership.example "<group-uuid>/<membership-uuid>"
```

Example:

```bash
terraform import snyk_group_membership.member "b667f176-df52-4b0a-9954-117af6b05ab7/550e8400-e29b-41d4-a716-446655440000"
```

Imported state sets `cascade_delete` to `false`; adjust in configuration if you need a different value on delete.

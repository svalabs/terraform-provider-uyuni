---
page_title: "Provider: Uyuni"
subcategory: ""
description: |-
  Use the Uyuni provider to manage users, roles, and other resources in Uyuni/SUSE Manager.
---

# Uyuni Provider

Use the Uyuni provider to manage users, roles, and other resources in [Uyuni](https://uyuni-project.org) and [SUSE Manager](https://www.suse.com/products/suse-manager/) through infrastructure as code.

The provider uses the Uyuni XML-RPC API to perform operations and requires administrative access to the Uyuni server.

## Example Usage

```terraform
terraform {
  required_providers {
    uyuni = {
      source = "registry.terraform.io/svalabs/uyuni"
    }
  }
}

provider "uyuni" {
  host     = "https://uyuni-server.example.com"
  username = "admin"
  password = "admin-password"
}

# Create a user with roles
resource "uyuni_user" "admin_user" {
  login     = "newadmin"
  firstname = "New"
  lastname  = "Admin"
  email     = "newadmin@example.com"
  password  = "SecurePassword123!"
  roles     = ["org_admin", "system_group_admin"]
}

# List all users
data "uyuni_users" "all" {}

output "user_count" {
  value = length(data.uyuni_users.all.users)
}
```

## Authentication

The Uyuni provider supports multiple authentication methods:

### Static Credentials

```terraform
provider "uyuni" {
  host     = "https://uyuni-server.example.com"
  username = "admin"
  password = "password"
}
```

### Environment Variables

```terraform
provider "uyuni" {
  # Configuration will be read from environment variables:
  # UYUNI_HOST, UYUNI_USERNAME, UYUNI_PASSWORD
}
```

Set the environment variables:
```bash
export UYUNI_HOST="https://uyuni-server.example.com"
export UYUNI_USERNAME="admin"
export UYUNI_PASSWORD="password"
```

### Terraform Variables

```terraform
variable "uyuni_host" {
  description = "Uyuni server URL"
  type        = string
}

variable "uyuni_username" {
  description = "Uyuni username"
  type        = string
}

variable "uyuni_password" {
  description = "Uyuni password"
  type        = string
  sensitive   = true
}

provider "uyuni" {
  host     = var.uyuni_host
  username = var.uyuni_username
  password = var.uyuni_password
}
```

## Schema

### Optional

- `host` (String) Uyuni server URL (e.g., https://uyuni-server.example.com). Can also be set with the `UYUNI_HOST` environment variable.
- `username` (String) Uyuni username with administrative privileges. Can also be set with the `UYUNI_USERNAME` environment variable.
- `password` (String, Sensitive) Uyuni user password. Can also be set with the `UYUNI_PASSWORD` environment variable.

## Requirements

### Server Requirements
- **Uyuni** 2022.12 or later
- **SUSE Manager** 4.3 or later
- XML-RPC API enabled (default)
- Administrative user account

### Network Requirements
- HTTPS access to Uyuni server (typically port 443)
- Valid SSL certificate or certificate acceptance configured
- Network connectivity from Terraform execution environment

### User Permissions
The authenticating user must have:
- Administrative privileges in the Uyuni organization
- Permission to create, modify, and delete users
- Permission to assign and remove user roles

## Available Resources and Data Sources

### Resources
- [`uyuni_user`](resources/user) - Manage users and their role assignments

### Data Sources
- [`uyuni_users`](data-sources/users) - List all users in the organization

## Common Use Cases

### User Lifecycle Management
```terraform
# Create users for different roles
resource "uyuni_user" "system_admin" {
  login     = "sysadmin"
  firstname = "System"
  lastname  = "Administrator"
  email     = "sysadmin@company.com"
  password  = "SecureAdminPass123!"
  roles     = ["org_admin"]
}

resource "uyuni_user" "operators" {
  for_each = toset(["ops1", "ops2", "ops3"])
  
  login     = each.key
  firstname = "Operator"
  lastname  = title(each.key)
  email     = "${each.key}@company.com"
  password  = "OperatorPass123!"
  roles     = ["system_group_admin"]
}
```

### Import Existing Users
```terraform
# Import existing users into Terraform management
resource "uyuni_user" "existing_admin" {
  login     = "admin"
  firstname = "Administrator"
  lastname  = "User"
  email     = "admin@company.com"
  password  = "new_secure_password"
  roles     = ["org_admin"]
}
```

Import command:
```bash
terraform import uyuni_user.existing_admin admin
```

### Conditional User Creation
```terraform
data "uyuni_users" "existing" {}

locals {
  existing_logins = [for user in data.uyuni_users.existing.users : user.login]
  create_backup_admin = !contains(local.existing_logins, "backup-admin")
}

resource "uyuni_user" "backup_admin" {
  count = local.create_backup_admin ? 1 : 0
  
  login     = "backup-admin"
  firstname = "Backup"
  lastname  = "Administrator"
  email     = "backup@company.com"
  password  = "BackupAdminPass123!"
  roles     = ["org_admin"]
}
```

## Error Handling

The provider includes comprehensive error handling and validation:

### Authentication Errors
```
Error: Unable to Create Uyuni API Client
authentication failed: invalid credentials
```

**Solutions:**
- Verify username and password are correct
- Check that the user has administrative privileges
- Ensure the Uyuni server is accessible

### Role Assignment Errors
```
Error: Could not add role to user
Invalid role: invalid_role_name
```

**Solutions:**
- Use valid role names (org_admin, system_group_admin, etc.)
- Check available roles in your Uyuni instance
- Verify user has permission to assign roles

### Import Errors
```
Error: User Not Found
Could not find user username for import
```

**Solutions:**
- Verify the username exists in Uyuni
- Check spelling and case sensitivity
- Ensure you have permission to view the user

## Logging and Debugging

Enable debug logging for troubleshooting:

```bash
export TF_LOG=DEBUG
export TF_LOG_PATH=terraform.log
terraform apply
```

The provider logs detailed information about:
- API requests and responses
- User creation and modification operations
- Role assignments and removals
- Import operations
- Error conditions and recovery

## Best Practices

### Security
1. **Use Environment Variables** for credentials in CI/CD pipelines
2. **Rotate Passwords Regularly** for Terraform service accounts
3. **Limit Permissions** to only what's necessary for Terraform operations
4. **Use HTTPS** for all Uyuni server connections

### User Management
1. **Import Existing Users** before managing them with Terraform
2. **Use Consistent Naming** conventions for user logins
3. **Document Role Assignments** and their purposes
4. **Regular Audits** of user accounts and permissions

### Terraform Configuration
1. **Use Variables** for sensitive and environment-specific values
2. **Organize Resources** logically with consistent naming
3. **Add Descriptions** to resources and variables
4. **Version Control** your Terraform configurations

## Limitations

### Current Limitations
- Password values cannot be retrieved from Uyuni (security feature)
- Role changes require the user to re-authenticate in some cases
- Bulk operations are performed sequentially (not parallelized)
- Limited to user and role management (other Uyuni resources planned)

### API Limitations
- XML-RPC API has rate limiting (typically not an issue for normal usage)
- Some operations require specific Uyuni versions or configurations
- Role availability depends on Uyuni/SUSE Manager edition and setup

## Support and Contributing

### Getting Help
1. **Documentation** - Check the provider documentation and examples
2. **Issues** - Search existing issues in the GitHub repository
3. **New Issues** - Create detailed bug reports or feature requests

### Contributing
Contributions are welcome! Please see the [contributing guidelines](https://github.com/svalabs/terraform-provider-uyuni/blob/main/CONTRIBUTING.md) for:
- Code standards and style
- Testing requirements
- Documentation updates
- Pull request process

### Development
For local development setup, see the [development guide](https://github.com/svalabs/terraform-provider-uyuni/blob/main/local-testing/local-dev-setup.md).

## Version Compatibility

| Component | Supported Versions |
|-----------|-------------------|
| Terraform | 1.0+ |
| Uyuni | 2022.12+ |
| SUSE Manager | 4.3+ |
| Go (development) | 1.21+ |

Always check the [release notes](https://github.com/svalabs/terraform-provider-uyuni/releases) for version-specific compatibility information.
---
page_title: "uyuni_user Resource - terraform-provider-uyuni"
subcategory: ""
description: |-
  Manages a user in Uyuni, including their roles and permissions.
---

# uyuni_user (Resource)

Manages a user in Uyuni/SUSE Manager, including their roles and permissions. This resource allows you to create, update, and delete users, as well as manage their role assignments.

## Example Usage

### Basic User Creation

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
  password = "admin"
}

resource "uyuni_user" "example" {
  login     = "jdoe"
  firstname = "John"
  lastname  = "Doe"
  email     = "jdoe@example.com"
  password  = "SecurePassword123!"
}
```

### User with Roles

```terraform
resource "uyuni_user" "admin_user" {
  login     = "newadmin"
  firstname = "New"
  lastname  = "Admin"
  email     = "newadmin@example.com"
  password  = "SecurePassword123!"
  roles     = ["org_admin", "system_group_admin"]
}
```

### User with Multiple Roles

```terraform
resource "uyuni_user" "power_user" {
  login     = "poweruser"
  firstname = "Power"
  lastname  = "User"
  email     = "power@example.com"
  password  = "PowerUserPass123!"
  roles     = [
    "system_group_admin",
    "channel_admin",
    "config_admin"
  ]
}
```

### Using Environment Variables

```terraform
provider "uyuni" {
  # Uses UYUNI_HOST, UYUNI_USERNAME, UYUNI_PASSWORD environment variables
}

resource "uyuni_user" "env_user" {
  login     = "envuser"
  firstname = "Environment"
  lastname  = "User"
  email     = "env@example.com"
  password  = var.user_password
  roles     = ["system_group_admin"]
}
```

## Schema

### Required

- `email` (String) User's email address
- `firstname` (String) User's first name
- `lastname` (String) User's last name
- `login` (String) User's login name (must be unique)
- `password` (String, Sensitive) User's password

### Optional

- `roles` (Set of String) Set of roles assigned to the user. Common roles include: org_admin, system_group_admin, channel_admin, config_admin.

## Available Roles

The following roles are commonly available in Uyuni/SUSE Manager:

- **`org_admin`** - Organization administrator with full permissions
- **`system_group_admin`** - Can manage system groups and their systems
- **`channel_admin`** - Can manage software channels (repositories)
- **`config_admin`** - Can manage configuration channels and files
- **`activation_key_admin`** - Can manage activation keys
- **`monitoring_admin`** - Can manage monitoring configurations (SUSE Manager)
- **`image_admin`** - Can manage container and OS images (SUSE Manager)

**Note:** Available roles may vary depending on your Uyuni version and configuration. Use the `uyuni_users` data source to check existing users and their role assignments.

## Import

Users can be imported using their login name:

```shell
terraform import uyuni_user.example username
```

### Import Behavior

When importing a user:
- All user details (first name, last name, email) are imported from Uyuni
- All assigned roles are imported and will be managed by Terraform
- The password field will be set to a placeholder value and must be updated in your configuration
- The imported user will be fully managed by Terraform after import

### Example Import Workflow

1. Import the existing user:
```shell
terraform import uyuni_user.existing_admin admin_user
```

2. Add the user configuration to your Terraform file:
```terraform
resource "uyuni_user" "existing_admin" {
  login     = "admin_user"
  firstname = "Admin"      # Must match existing user
  lastname  = "User"       # Must match existing user  
  email     = "admin@example.com"  # Must match existing user
  password  = "new_password"       # Set new password
  roles     = ["org_admin"]        # Current roles will be imported
}
```

3. Run `terraform plan` to see what changes will be made (typically just password update)

4. Apply the configuration:
```shell
terraform apply
```

## Role Management

### Adding Roles
```terraform
resource "uyuni_user" "user_with_new_roles" {
  login     = "existing_user"
  firstname = "Existing"
  lastname  = "User"
  email     = "existing@example.com"
  password  = "password123"
  roles     = ["system_group_admin", "channel_admin"]  # Add roles here
}
```

### Removing Roles
```terraform
resource "uyuni_user" "user_remove_roles" {
  login     = "existing_user"
  firstname = "Existing"
  lastname  = "User"
  email     = "existing@example.com"
  password  = "password123"
  roles     = ["system_group_admin"]  # Remove unwanted roles
}
```

### No Role Management
```terraform
resource "uyuni_user" "user_no_roles" {
  login     = "basic_user"
  firstname = "Basic"
  lastname  = "User"
  email     = "basic@example.com"
  password  = "password123"
  roles     = []  # No roles assigned
}
```

## Notes

- When a user is deleted, all their role assignments are automatically removed
- Role changes are applied immediately and may affect the user's permissions
- The password field is sensitive and will not be displayed in Terraform output
- After importing, the password field must be specified in the configuration as it cannot be retrieved from the API
- Import preserves all existing role assignments - they will be managed by Terraform after import
- User login names must be unique within the Uyuni organization
- Email addresses should be valid and are used for notifications

## Error Handling

### Common Errors

**User Already Exists:**
```
Error: Could not create user: User already exists
```
Solution: Use `terraform import` to import the existing user instead.

**Invalid Role:**
```
Error: Could not add role to user: Invalid role
```
Solution: Check that the role exists in your Uyuni instance and that you have permission to assign it.

**Permission Denied:**
```
Error: Could not create user: Permission denied
```
Solution: Ensure your Uyuni user has administrative privileges to manage users.

## Best Practices

1. **Use Strong Passwords:** Always use strong, unique passwords for user accounts
2. **Principle of Least Privilege:** Only assign roles that are necessary for the user's responsibilities
3. **Regular Audits:** Periodically review user roles and permissions
4. **Import Existing Users:** Use import functionality to bring existing users under Terraform management
5. **Environment Variables:** Use environment variables for sensitive provider configuration
6. **Role Validation:** Test role assignments in a development environment before applying to production

## Related Resources

- `uyuni_users` - Data source to list all users in the organization
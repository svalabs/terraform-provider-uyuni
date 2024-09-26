terraform {
  required_providers {
    uyuni = {
      source = "registry.terraform.io/svalabs/uyuni"
    }
  }
}

provider "uyuni" {
    host = "192.168.1.100"
    username = "admin"
    password = "admin"
}

data "uyuni_users" "my_users" {}

resource "uyuni_user" "sgiertz" {
  login = "sgiertz"
  firstname = "Simone"
  lastname = "Giertz"
  email = "sgiertz@foo.bar"
  password = "test123"
}

output "users" {
  value = data.uyuni_users.my_users
}

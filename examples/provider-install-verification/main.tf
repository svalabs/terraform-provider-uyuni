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

# data "uyuni_users" "example" {}

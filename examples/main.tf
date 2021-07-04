terraform {
  required_providers {
    vyos = {
      version = "0.1.0"
      source  = "foltik/vyos"
    }
  }
}

provider "vyos" {
  url = "https://vyos.local"
  key = "XXXXXXXX"
}

resource "vyos_config" "host_name" {
  key   = "system host_name"
  value = "vyos"
}

output "host_name" {
  value = vyos_config.host_name.value
}

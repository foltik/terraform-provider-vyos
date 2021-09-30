terraform {
  required_providers {
    vyos = {
      version = "0.x.x"
      source  = "foltik/vyos"
    }
  }
}

provider "vyos" {
  url = "https://vyos.local"
  key = "xxxxxxxxx"
}

# Equivalent to "set system host-name vyos"
resource "vyos_config" "hostname" {
  key   = "system host-name"
  value = "vyos"
}

# Equivalent to "set system static-host-mapping host-name test.local inet 10.0.0.1"
resource "vyos_static_host_mapping" "mapping" {
  host = "test.local"
  ip = "10.0.0.1"
}

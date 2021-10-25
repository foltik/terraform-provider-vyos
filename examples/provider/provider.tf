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

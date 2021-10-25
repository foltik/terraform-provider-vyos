resource "vyos_config_block" "allow_sith_supremacy" {
  path = "firewall name Empire-Senate rule 66"

  configs = {
    "action"      = "accept"
    "description" = "For a safe and secure society"
    "log"         = "disable"
  }
}

resource "vyos_config_block" "allow_sith_supremacy_jedi" {
  path = "firewall name Empire-Senate rule 66 destination group"

  configs = {
    "port-group" = "Jedi"
  }

  depends_on = [vyos_config_block.allow_sith_supremacy]
}

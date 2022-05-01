resource "vyos_config_block" "allow_sith_supremacy" {
  path = "firewall name Empire-Senate rule 66"

  configs = {
    "action"      = "accept"
    "description" = "For a safe and secure society"
    "log"         = "disable"
    "destination group port-group": "Jedi"
  }
}
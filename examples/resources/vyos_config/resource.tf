# Performs "set system host-name vyos"
resource "vyos_config" "hostname" {
  key   = "system host-name"
  value = "vyos"
}

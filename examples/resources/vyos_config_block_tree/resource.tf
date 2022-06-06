resource "vyos_config_block" "allow_sith_supremacy" {
  path = "firewall name Empire-Senate rule 66"

  configs = {
    "action"      = "accept"
    "description" = "For a safe and secure society"
    "log"         = "disable"
    "destination group port-group": "Jedi"
  }
}

resource "vyos_config_block_tree" "ssh" {
  path = "service ssh"

  configs = {
    "port"      = "22",
    "disable-password-authentication" = "", #Keep simple passwords for login via terminal but require key for ssh
    "listen-address" = jsonencode(["192.168.2.1", "192.168.63.1"]) # Listen in LAN and management interface
  }
}

# Equivalent to "set system static-host-mapping host-name test.local inet 10.0.0.1"
resource "vyos_static_host_mapping" "mapping" {
  host = "test.local"
  ip   = "10.0.0.1"
}

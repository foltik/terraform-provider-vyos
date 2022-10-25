---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vyos_interface_ethernet Resource - terraform-provider-vyos"
subcategory: ""
description: |-
  This will be the most widely used interface on a router carrying traffic to the real world. VyOS doc https://docs.vyos.io/en/latest/configuration/interfaces/ethernet.html.
---

# vyos_interface_ethernet (Resource)

This will be the most widely used interface on a router carrying traffic to the real world. [VyOS doc](https://docs.vyos.io/en/latest/configuration/interfaces/ethernet.html).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Ethernet interface name. Eg: eth0

### Optional

- `address` (List of String) `<x.x.x.x/x | h:h:h:h:h:h:h:h/x | dhcp | dhcpv6>` Configure interface <interface> with one or more interface addresses.
- `description` (String) Group description text.
- `dhcp_options` (Block List, Max: 1) DHCP client settings/options (see [below for nested schema](#nestedblock--dhcp_options))
- `dhcpv6_options` (Block List, Max: 1) DHCPv6 client settings/options (see [below for nested schema](#nestedblock--dhcpv6_options))
- `disable` (Boolean) Administratively disable interface.
- `disable_flow_control` (Boolean) Disable Ethernet flow control (pause frames)
- `disable_link_detect` (Boolean) Ignore link state changes.
- `duplex` (String) Duplex mode.
- `eapol` (Block List, Max: 1) Extensible Authentication Protocol over Local Area Network (see [below for nested schema](#nestedblock--eapol))
- `firewall` (Block List, Max: 1) Firewall options. [Interface base firewall](https://docs.vyos.io/en/latest/configuration/firewall/general.html#applying-a-rule-set-to-an-interface) (see [below for nested schema](#nestedblock--firewall))
- `hw_id` (String) Associate Ethernet Interface with given Media Access Control (MAC) address.
- `ip` (Block List, Max: 1) IPv4 routing parameters. (see [below for nested schema](#nestedblock--ip))
- `ipv6` (Block List, Max: 1) ipv6 routing parameters. (see [below for nested schema](#nestedblock--ipv6))
- `mac` (String) Configure user defined MAC address on given <interface>. Media Access Control (MAC) address.
- `mirror` (Block List, Max: 1) Incoming/outgoing packet mirroring destination. (see [below for nested schema](#nestedblock--mirror))
- `mtu` (Number) Maximum Transmission Unit (MTU).
- `offload` (Block List, Max: 1) Configurable offload options. (see [below for nested schema](#nestedblock--offload))
- `policy` (Block List, Max: 1) Policy route options (see [below for nested schema](#nestedblock--policy))
- `redirect` (String) Incoming packet redirection destination, interface for packet redirection
- `ring_buffer` (Block List, Max: 1) Shared buffer between the device driver and NIC (see [below for nested schema](#nestedblock--ring_buffer))
- `speed` (String) Link speed
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `traffic_policy` (Block List, Max: 1) Traffic-policy for interface (see [below for nested schema](#nestedblock--traffic_policy))
- `vrf` (String) VRF instance name
- `xdp` (Boolean) Enable eXpress Data Path

### Read-Only

- `id` (String) The resource ID

<a id="nestedblock--dhcp_options"></a>
### Nested Schema for `dhcp_options`

Optional:

- `client_id` (String) Identifier used by client to identify itself to the DHCP server. [RFC 2131](https://datatracker.ietf.org/doc/html/rfc2131.html) states: The client MAY choose to explicitly provide the identifier through the ‘client identifier’ option. If the client supplies a ‘client identifier’, the client MUST use the same ‘client identifier’ in all subsequent messages, and the server MUST use that identifier to identify the client.
- `default_route_distance` (Number) Distance for the default route from DHCP server.
- `host_name` (String) Override system host-name sent to DHCP server.
- `no_default_route` (Boolean) Do not request routers from DHCP server.
- `reject` (List of String) `<x.x.x.x | x.x.x.x/x>` IP addresses or subnets from which to reject DHCP leases.
- `vendor_class_id` (String) Identify the vendor client type to the DHCP server.


<a id="nestedblock--dhcpv6_options"></a>
### Nested Schema for `dhcpv6_options`

Optional:

- `duid` (String) DHCP unique identifier (DUID) to be sent by dhcpv6 client.
- `parameters_only` (Boolean) Acquire only config parameters, no address.
- `rapid_commit` (Boolean) Wait for immediate reply instead of advertisements.
- `temporary` (Boolean) IPv6 temporary address. Request only a temporary address and not form an IA_NA (Identity Association for Non-temporary Addresses) partnership.


<a id="nestedblock--eapol"></a>
### Nested Schema for `eapol`

Optional:

- `ca_certificate` (String) Certificate Authority in PKI configuration.
- `certificate` (String) Certificate in PKI configuration.
- `passphrase` (String) Private key passphrase.


<a id="nestedblock--firewall"></a>
### Nested Schema for `firewall`

Optional:

- `in` (Block List, Max: 1) Ruleset for forwarded packets on inbound interface. (see [below for nested schema](#nestedblock--firewall--in))
- `local` (Block List, Max: 1) Ruleset for packets destined for this router. (see [below for nested schema](#nestedblock--firewall--local))
- `out` (Block List, Max: 1) Ruleset for forwarded packets on outbound interface. (see [below for nested schema](#nestedblock--firewall--out))

<a id="nestedblock--firewall--in"></a>
### Nested Schema for `firewall.in`

Optional:

- `ipv6_name` (String) Inbound IPv6 firewall ruleset name for interface.
- `name` (String) Inbound IPv4 firewall ruleset name for interface.


<a id="nestedblock--firewall--local"></a>
### Nested Schema for `firewall.local`

Optional:

- `ipv6_name` (String) Inbound IPv6 firewall ruleset name for interface.
- `name` (String) Inbound IPv4 firewall ruleset name for interface.


<a id="nestedblock--firewall--out"></a>
### Nested Schema for `firewall.out`

Optional:

- `ipv6_name` (String) Inbound IPv6 firewall ruleset name for interface.
- `name` (String) Inbound IPv4 firewall ruleset name for interface.



<a id="nestedblock--ip"></a>
### Nested Schema for `ip`

Optional:

- `adjust_mss` (String) `<clamp-mss-to-pmtu | 500-65535>`. _Automatically sets the MSS to the proper value_, or _TCP Maximum segment size in bytes_
- `arp_cache_timeout` (Number) ARP cache entry timeout in seconds.
- `disable_arp_filter` (Boolean) Disable ARP filter on this interface.
- `disable_forwarding` (Boolean) Disable IP forwarding on this interface.
- `enable_arp_accept` (Boolean) Enable ARP accept on this interface.
- `enable_arp_announce` (Boolean) Enable ARP announce on this interface.
- `enable_arp_ignore` (Boolean) Enable ARP ignore on this interface.
- `enable_proxy_arp` (Boolean) Enable proxy-arp on this interface.
- `proxy_arp_pvlan` (Boolean) Enable private VLAN proxy ARP on this interface.
- `source_validation` (String) `<strict | loose | disable>`. Enable policy for source validation by reversed path, as specified in RFC [3704](https://datatracker.ietf.org/doc/html/rfc3704.html). Current recommended practice in RFC 3704 is to enable strict mode to prevent IP spoofing from DDos attacks. If using asymmetric routing or other complicated routing, then loose mode is recommended. [VyOS doc](https://docs.vyos.io/en/latest/configuration/interfaces/ethernet.html#cfgcmd-set-interfaces-ethernet-interface-ip-source-validation-strict-loose-disable)


<a id="nestedblock--ipv6"></a>
### Nested Schema for `ipv6`

Optional:

- `address` (Block List, Max: 1) IPv6 address configuration modes (see [below for nested schema](#nestedblock--ipv6--address))
- `adjust_mss` (String) `<clamp-mss-to-pmtu | 500-65535>`. _Automatically sets the MSS to the proper value_, or _TCP Maximum segment size in bytes_
- `disable_forwarding` (Boolean) Disable IP forwarding on this interface.

<a id="nestedblock--ipv6--address"></a>
### Nested Schema for `ipv6.address`

Optional:

- `autoconf` (Boolean) Enable acquisition of IPv6 address using stateless autoconfig (SLAAC [RFC 4862](https://datatracker.ietf.org/doc/html/rfc4862.html)). __This method automatically disables IPv6 traffic forwarding on the interface in question.__
- `eui64` (List of String) `<h:h:h:h:h:h:h:h/64>` Prefix for IPv6 address with MAC-based EUI-64
- `no_default_link_local` (Boolean) Remove the default link-local address from the interface



<a id="nestedblock--mirror"></a>
### Nested Schema for `mirror`

Optional:

- `egress` (Set of String) `<ethX | ethX.Y | pethX | ...>` Mirror the egress traffic of the interface to the destination interface.
- `ingress` (Set of String) `<ethX | ethX.Y | pethX | ...>` Mirror the ingress traffic of the interface to the destination interface.


<a id="nestedblock--offload"></a>
### Nested Schema for `offload`

Optional:

- `gro` (Boolean) Enable Generic Receive Offload
- `gso` (Boolean) Enable Generic Segmentation Offload
- `lro` (Boolean) Enable Large Receive Offload
- `rps` (Boolean) Enable Receive Packet Steering
- `sg` (Boolean) Enable Scatter-Gather
- `tso` (Boolean) Enable TCP Segmentation Offloading


<a id="nestedblock--policy"></a>
### Nested Schema for `policy`

Optional:

- `ipv6_route` (String) IPv6 policy route ruleset for interface
- `route` (String) IPv4 policy route ruleset for interface


<a id="nestedblock--ring_buffer"></a>
### Nested Schema for `ring_buffer`

Optional:

- `rx` (Number) RX ring buffer size
- `tx` (Number) TX ring buffer size


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)


<a id="nestedblock--traffic_policy"></a>
### Nested Schema for `traffic_policy`

Optional:

- `in` (String) Ingress traffic policy for interface
- `out` (String) Egress traffic policy for interface


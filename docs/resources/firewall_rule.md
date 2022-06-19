---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "vyos_firewall_rule Resource - terraform-provider-vyos"
subcategory: ""
description: |-
  Firewall rules with criteria matching that can be applied to an interface or a zone, for more information see VyOS Firewall doc https://docs.vyos.io/en/latest/configuration/firewall/index.html#matching-criteria.
---

# vyos_firewall_rule (Resource)

Firewall rules with criteria matching that can be applied to an interface or a zone, for more information see [VyOS Firewall doc](https://docs.vyos.io/en/latest/configuration/firewall/index.html#matching-criteria).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **priority** (Number) _Must be unique within a rule set_. Data packets go through the rules based on the priority, from lowest to highest beginning at 0, at the first match the action of the rule will be applied and execution stops.
- **rule_set** (String) Rule set name this rule belongs to.

### Optional

- **action** (String) Action of this rule.
- **description** (String) Rule description text. Without a good description it can be hard to know why the rule exists.
- **destination** (Block List, Max: 1) Traffic destination match criteria. (see [below for nested schema](#nestedblock--destination))
- **disable** (Boolean) Disable this rule, but keep it in the config.
- **log** (String) Enable the logging of the this rule.
- **protocol** (String) Match a protocol criteria. A protocol number or a name which is defined in VyOS instances: `/etc/protocols` file. Special names are `all` for all protocols and `tcp_udp` for tcp and udp based packets. The `!` negate the selected protocol. `[<text> | <0-255> | all | tcp_udp]`
- **source** (Block List, Max: 1) Traffic source match criteria. (see [below for nested schema](#nestedblock--source))
- **state** (Block List, Max: 1) Match against the state of a packet. (see [below for nested schema](#nestedblock--state))
- **tcp** (Block List, Max: 1) TCP specific match criteria. (see [below for nested schema](#nestedblock--tcp))

### Read-Only

- **id** (String) The resource ID, same as the `priority`

<a id="nestedblock--destination"></a>
### Nested Schema for `destination`

Optional:

- **address** (String) Destination address to match against, can be in format of: `[<x.x.x.x> | <x.x.x.x>-<x.x.x.x> | <x.x.x.x/x>]`. By starting the field with a `!` it will be a negative match.
- **group** (Block Set) Use a pre-defined group. (see [below for nested schema](#nestedblock--destination--group))
- **mac_address** (String) Destination mac-address to match against. By starting the field with a `!` it will be a negative match.
- **port** (String) A port can be set with port number in format: `[<xx> | <xx>-<xx>]` or a name which is here defined: `/etc/services`. Multiple source ports can be specified as a comma-separated list. The whole list can also be “negated” using `!`.

<a id="nestedblock--destination--group"></a>
### Nested Schema for `destination.group`

Optional:

- **address_group** (String) Address group name.
- **network_group** (String) Network group name.
- **port_group** (String) Port group name.



<a id="nestedblock--source"></a>
### Nested Schema for `source`

Optional:

- **address** (String) Source address to match against, can be in format of: `[<x.x.x.x> | <x.x.x.x>-<x.x.x.x> | <x.x.x.x/x>]`. By starting the field with a `!` it will be a negative match.
- **group** (Block Set) Use a pre-defined group. (see [below for nested schema](#nestedblock--source--group))
- **mac_address** (String) Source mac-address to match against. By starting the field with a `!` it will be a negative match.
- **port** (String) A port can be set with port number in format: `[<xx> | <xx>-<xx>]` or a name which is here defined: `/etc/services`. Multiple source ports can be specified as a comma-separated list. The whole list can also be “negated” using `!`.

<a id="nestedblock--source--group"></a>
### Nested Schema for `source.group`

Optional:

- **address_group** (String) Address group name.
- **network_group** (String) Network group name.
- **port_group** (String) Port group name.



<a id="nestedblock--state"></a>
### Nested Schema for `state`

Optional:

- **established** (String) If this rule should match against the connection state `established`, valied values: `[enable | disable]`
- **invalid** (String) If this rule should match against the connection state `invalid`, valied values: `[enable | disable]`
- **new** (String) If this rule should match against the connection state `new`, valied values: `[enable | disable]`
- **related** (String) If this rule should match against the connection state `related`, valied values: `[enable | disable]`


<a id="nestedblock--tcp"></a>
### Nested Schema for `tcp`

Optional:

- **flags** (String) Allowed values for TCP flags: `SYN`, `ACK`, `FIN`, `RST`, `URG`, `PSH`, `ALL` When specifying more than one flag, flags should be comma separated. The `!` negate the selected protocol.


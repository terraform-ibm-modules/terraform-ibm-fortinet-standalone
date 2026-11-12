data "ibm_is_vpc" "vpc1" {
  name = var.VPC
}
data "ibm_is_subnet" "subnet1" {
  identifier = var.SUBNET1
}
data "ibm_is_subnet" "subnet2" {
  identifier = var.SUBNET2
}

# ============================================================================
# Security Group - Public (port1 external)
# Inbound : deny all (no rules = implicit deny in IBM VPC)
# Outbound: restricted to FortiGuard services for licensing/DNS/updates
# ============================================================================

resource "ibm_is_security_group" "fgt_sg_public" {
  name           = "${var.CLUSTER_NAME}-sg-public-${random_string.random_suffix.result}"
  vpc            = data.ibm_is_vpc.vpc1.id
  resource_group = data.ibm_resource_group.rg.id
}

# Outbound - UDP 53 for FortiGuard DNS & SDNS queries
resource "ibm_is_security_group_rule" "fgt_public_outbound_dns" {
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
  protocol  = "udp"
  port_min  = 53
  port_max  = 53
  name      = "allow-outbound-fortiguard-dns-udp-53"
}

# Outbound - TCP 853 for DNS over TLS (DoT)
resource "ibm_is_security_group_rule" "fgt_public_outbound_dot" {
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
  protocol  = "tcp"
  port_min  = 853
  port_max  = 853
  name      = "allow-outbound-dns-over-tls-tcp-853"
}

# Outbound - TCP 443 for Licensing, FortiCare & Entitlements
resource "ibm_is_security_group_rule" "fgt_public_outbound_https" {
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
  protocol  = "tcp"
  port_min  = 443
  port_max  = 443
  name      = "allow-outbound-fortiguard-licensing-tcp-443"
}

# Outbound - TCP 8890 for FortiGuard Distribution Updates (optional)
resource "ibm_is_security_group_rule" "fgt_public_outbound_fortiguard_updates" {
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
  protocol  = "tcp"
  port_min  = 8890
  port_max  = 8890
  name      = "allow-outbound-fortiguard-updates-tcp-8890"
}

# Outbound - UDP 123 for NTP time synchronization
resource "ibm_is_security_group_rule" "fgt_public_outbound_ntp" {
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "outbound"
  remote    = "0.0.0.0/0"
  protocol  = "udp"
  port_min  = 123
  port_max  = 123
  name      = "allow-outbound-ntp-udp-123"
}

# Inbound - TCP 22 for CI SSH init (disclaimer acceptance + password change on first boot)
# Only created when CI_MODE=true — not present in production deployments
# remote is locked to the pipeline worker's IP (CI_ALLOWED_CIDR) — never 0.0.0.0/0
resource "ibm_is_security_group_rule" "fgt_public_inbound_ci_ssh" {
  count     = var.CI_MODE ? 1 : 0
  group     = ibm_is_security_group.fgt_sg_public.id
  direction = "inbound"
  remote    = var.CI_ALLOWED_CIDR
  protocol  = "tcp"
  port_min  = 22
  port_max  = 22
  name      = "allow-inbound-ci-mgmt-tcp-22"
}



# ============================================================================
# Security Group - Private (port2 internal)
# Inbound : deny all (no rules = implicit deny in IBM VPC)
# Outbound: self-referencing only — no internet access needed on internal port
# ============================================================================

resource "ibm_is_security_group" "fgt_sg_private" {
  name           = "${var.CLUSTER_NAME}-sg-private-${random_string.random_suffix.result}"
  vpc            = data.ibm_is_vpc.vpc1.id
  resource_group = data.ibm_resource_group.rg.id
}

# Outbound - allow traffic only to interfaces in this same SG
resource "ibm_is_security_group_rule" "fgt_private_outbound_self" {
  group     = ibm_is_security_group.fgt_sg_private.id
  direction = "outbound"
  remote    = ibm_is_security_group.fgt_sg_private.id
  name      = "allow-outbound-to-same-sg"
}

resource "ibm_is_virtual_network_interface" "vni-port1" {
  name                      = "${var.CLUSTER_NAME}-fgt-interface1-${random_string.random_suffix.result}"
  allow_ip_spoofing         = false
  auto_delete               = false
  enable_infrastructure_nat = true
  security_groups           = [ibm_is_security_group.fgt_sg_public.id]
  subnet                    = data.ibm_is_subnet.subnet1.id
}

resource "ibm_is_virtual_network_interface" "vni-port2" {
  name                      = "${var.CLUSTER_NAME}-fgt-interface2-${random_string.random_suffix.result}"
  allow_ip_spoofing         = false
  auto_delete               = false
  enable_infrastructure_nat = true
  security_groups           = [ibm_is_security_group.fgt_sg_private.id]
  subnet                    = data.ibm_is_subnet.subnet2.id
}
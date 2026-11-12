data "ibm_is_ssh_key" "ssh_key" {
  name = var.SSH_PUBLIC_KEY
}

// Log disk volumes are only provisioned for profiles lacking local instance storage (e.g. cx2).
// cx3d profiles already include built-in local NVMe instance storage (/dev/vdb) used for disk logging;
// attaching an extra block volume creates a redundant disk on cx3d instances.
resource "ibm_is_volume" "logDisk" {
  count = startswith(local.selected_profile, "cx3d") ? 0 : 1
  // Name must be lower case
  name    = "${var.CLUSTER_NAME}-logdisk-${random_string.random_suffix.result}"
  profile = "10iops-tier"
  zone    = var.ZONE1
}

resource "ibm_is_floating_ip" "publicip" {
  name = "${var.CLUSTER_NAME}-publicip-${random_string.random_suffix.result}"
  zone = var.ZONE1
}

resource "ibm_is_virtual_network_interface_floating_ip" "public_ip" {
  virtual_network_interface = ibm_is_virtual_network_interface.vni-port1.id
  floating_ip               = ibm_is_floating_ip.publicip.id
}

resource "ibm_is_instance" "fgt1" {
  name    = "${var.CLUSTER_NAME}-licensed-fortigate-${random_string.random_suffix.result}"
  profile = local.selected_profile
  vpc     = data.ibm_is_vpc.vpc1.id
  zone    = var.ZONE1

  catalog_offering {
    version_crn = var.CATALOG_OFFERING_VERSION_CRN
    plan_crn    = var.CATALOG_OFFERING_PLAN_CRN
  }

  primary_network_attachment {
    name = "${var.CLUSTER_NAME}-fgt-port1-${random_string.random_suffix.result}"
    virtual_network_interface {
      id = ibm_is_virtual_network_interface.vni-port1.id
    }
  }
  network_attachments {
    name = "${var.CLUSTER_NAME}-fgt-port2-${random_string.random_suffix.result}"
    virtual_network_interface {
      id = ibm_is_virtual_network_interface.vni-port2.id
    }
  }

  // Only attach log volume for profiles without instance storage (cx2); cx3d uses built-in NVMe
  volumes            = startswith(local.selected_profile, "cx3d") ? [] : [ibm_is_volume.logDisk[0].id]
  auto_delete_volume = true
  user_data          = file(var.USER_DATA)
  keys               = [data.ibm_is_ssh_key.ssh_key.id]

  # Allocate 1 Gbps (1000 Mbps) to volumes, rest goes to networking
  total_volume_bandwidth = 1000

  # Enable IBM Cloud metadata service for VNF licensing
  metadata_service {
    enabled            = true
    protocol           = "https"
    response_hop_limit = 5
  }
}



// Your SSH key
variable "SSH_PUBLIC_KEY" {
  default     = ""
  description = "Public ssh ID name. This needs to be pre-created."
}
// Magic Value for Calalog Validation that initlizes terraform with a specific version.
// Only needed in IBM catalog.
variable "TF_VERSION" {
  default     = "1.6.6"
  description = "Terraform version to be used in validation"
}
// IBM Regions
variable "REGION" {
  type        = string
  default     = "us-south"
  description = "Deployment Region"

}

// IBM availability zones
variable "ZONE1" {
  type        = string
  default     = "us-south-1"
  description = "Deployment Zone."

}
variable "VPC" {
  type        = string
  default     = ""
  description = "Name of the VPC you want to deploy a FortiGate into."
}
variable "SUBNET1" {
  type        = string
  default     = ""
  description = "The ID of the Primary, Public Subnet Used for port1 on the FortiGate"
}
variable "SUBNET2" {
  type        = string
  default     = ""
  description = "The ID of the Secondary, Private Subnet Used for port2 on the FortiGate"
}

variable "RESOURCE_GROUP" {
  type        = string
  default     = "Default"
  description = "The RESOURCE Group Name to attach to the FortiGate instance."
}

data "ibm_resource_group" "rg" {
  name = var.RESOURCE_GROUP
}

// Name will be in the format of CLUSTER_NAME-RESOURCE-randomSuffix to be easily identifiable.
// Name must be lowercase
variable "CLUSTER_NAME" {
  type        = string
  default     = "fortigate-terraform"
  description = "Cluster name will be appended by a random Suffix to prevent collisions and allow easier identification."

}
// Random Suffix to avoid name collisions and identify cluster.
resource "random_string" "random_suffix" {
  length           = 4
  special          = true
  override_special = ""
  min_lower        = 4
}


// Specify the Version CRN for the Catalog offering representing the Licensed Fortinet Standalone Firewall.
variable "CATALOG_OFFERING_VERSION_CRN" {
  default     = ""
  description = "IBM Cloud Licensed Firewall Version CRN"
}

// Specify the Plan CRN for the Catalog offering representing the Licensed Fortinet Standalone Firewall.
variable "CATALOG_OFFERING_PLAN_CRN" {
  default     = ""
  description = "IBM Cloud Licensed Firewall Plan CRN"
}

//For more details see: https://cloud.ibm.com/docs/account?topic=account-userapikey
variable "IBMCLOUD_API_KEY" {
  default     = ""
  description = "IBM Gen2 API key."
}
// Bootstrap configuration file
variable "USER_DATA" {
  type        = string
  default     = "user_data.conf"
  description = "The Custom Bootstrap Data file name."
}

variable "CI_MODE" {
  type        = bool
  default     = false
  description = "When true, adds inbound HTTPS (443) rule to public security group for CI acceptance testing."
}

variable "CI_ALLOWED_CIDR" {
  type        = string
  default     = ""
  description = "CIDR block allowed to reach the FortiGate on ports 22 and 443 during CI. Set to the pipeline worker's public IP as a /32 (e.g. '169.46.1.2/32'). Only used when CI_MODE=true."
}

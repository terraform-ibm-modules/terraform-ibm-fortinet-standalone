output "FortiGate_Public_IP" {
  value = ibm_is_floating_ip.publicip.address
}

output "Username" {
  value = "admin"
}

output "Default_Admin_Password" {
  value = ibm_is_instance.fgt1.id
}

output "CATALOG_OFFERING_VERSION_CRN" {
  description = "IBM Cloud Licensed Firewall Version CRN used for deployment"
  value       = var.CATALOG_OFFERING_VERSION_CRN
}

output "CATALOG_OFFERING_PLAN_CRN" {
  description = "IBM Cloud Licensed Firewall Plan CRN used for deployment"
  value       = var.CATALOG_OFFERING_PLAN_CRN
}

output "selected_vsi_profile" {
  description = "VSI Profile selected based on the plan CRN mapping"
  value       = local.selected_profile
}

output "Security_Group_Public_ID" {
  description = "ID of the public security group (port1 - external)"
  value       = ibm_is_security_group.fgt_sg_public.id
}

output "Security_Group_Public_Name" {
  description = "Name of the public security group (port1 - external)"
  value       = ibm_is_security_group.fgt_sg_public.name
}

output "Security_Group_Private_ID" {
  description = "ID of the private security group (port2 - internal)"
  value       = ibm_is_security_group.fgt_sg_private.id
}

output "Security_Group_Private_Name" {
  description = "Name of the private security group (port2 - internal)"
  value       = ibm_is_security_group.fgt_sg_private.name
}
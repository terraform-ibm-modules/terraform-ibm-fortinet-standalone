#!/usr/bin/env bash
# CI environment loader for FortiGate Standalone
# Source this file — do NOT execute it directly.
# Sets all TF_VAR_* exports and IC_API_KEY needed by ci-deploy.sh and ci-destroy.sh.
#
# Required pipeline properties (all individual get_env, except API key):
#   TF_VAR_REGION, TF_VAR_ZONE1, TF_VAR_VPC, TF_VAR_SUBNET1, TF_VAR_SUBNET2,
#   TF_VAR_CATALOG_OFFERING_VERSION_CRN, TF_VAR_CATALOG_OFFERING_PLAN_CRN,
#   TF_VAR_SSH_PUBLIC_KEY   — text properties (get_env)
#   TF_VAR_IBMCLOUD_API_KEY — secure property (get_secret)

export TF_VAR_RESOURCE_GROUP="${TF_VAR_RESOURCE_GROUP:-$(get_env TF_VAR_RESOURCE_GROUP "default")}"
export TF_VAR_REGION="${TF_VAR_REGION:-$(get_env TF_VAR_REGION "")}"
export TF_VAR_ZONE1="${TF_VAR_ZONE1:-$(get_env TF_VAR_ZONE1 "")}"
export TF_VAR_VPC="${TF_VAR_VPC:-$(get_env TF_VAR_VPC "")}"
export TF_VAR_SUBNET1="${TF_VAR_SUBNET1:-$(get_env TF_VAR_SUBNET1 "")}"
export TF_VAR_SUBNET2="${TF_VAR_SUBNET2:-$(get_env TF_VAR_SUBNET2 "")}"
export TF_VAR_CATALOG_OFFERING_VERSION_CRN="${TF_VAR_CATALOG_OFFERING_VERSION_CRN:-$(get_env TF_VAR_CATALOG_OFFERING_VERSION_CRN "")}"
export TF_VAR_CATALOG_OFFERING_PLAN_CRN="${TF_VAR_CATALOG_OFFERING_PLAN_CRN:-$(get_env TF_VAR_CATALOG_OFFERING_PLAN_CRN "")}"
export TF_VAR_SSH_PUBLIC_KEY="${TF_VAR_SSH_PUBLIC_KEY:-$(get_env TF_VAR_SSH_PUBLIC_KEY "")}"

# Secure property — must use get_secret, not get_env
export TF_VAR_IBMCLOUD_API_KEY="${TF_VAR_IBMCLOUD_API_KEY:-$(get_secret TF_VAR_IBMCLOUD_API_KEY "")}"

# Override IC_API_KEY so the IBM Terraform provider uses the deployment key,
# not the OnePipeline service ID key injected into the container.
export IC_API_KEY="$TF_VAR_IBMCLOUD_API_KEY"

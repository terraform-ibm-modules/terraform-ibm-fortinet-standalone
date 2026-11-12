#!/usr/bin/env bash
# CI Pipeline - Deploy Script for FortiGate Standalone
# Sourced by terraform-ci-deploy.sh in the deploy stage
# TERRAFORM_DIR and TF_OUTPUTS_PATH are set by the pipeline before sourcing

set -euo pipefail

echo "============================================"
echo "FortiGate Standalone - Deploy"
echo "============================================"

cd "$TERRAFORM_DIR"
echo "Working directory: $(pwd)"

# Load all TF_VAR_* exports and IC_API_KEY from the shared env loader
# shellcheck source=scripts/ci-env.sh
source "${TERRAFORM_DIR}/scripts/ci-env.sh"

# Validate required variables
REQUIRED_VARS=(
  TF_VAR_IBMCLOUD_API_KEY
  TF_VAR_REGION
  TF_VAR_ZONE1
  TF_VAR_VPC
  TF_VAR_SUBNET1
  TF_VAR_SUBNET2
  TF_VAR_CATALOG_OFFERING_VERSION_CRN
  TF_VAR_CATALOG_OFFERING_PLAN_CRN
  TF_VAR_SSH_PUBLIC_KEY
)
for var in "${REQUIRED_VARS[@]}"; do
  [ -z "${!var:-}" ] && echo "✗ Missing required variable: $var" && exit 1
done
echo "✓ All required variables present"

# Enable CI mode — opens inbound SSH (22) for acceptance tests
export TF_VAR_CI_MODE=true

# Lock CI inbound rules to this pipeline worker's public IP only (never 0.0.0.0/0)
PIPELINE_IP="$(curl -sf --max-time 10 https://checkip.amazonaws.com || curl -sf --max-time 10 https://api.ipify.org)"
if [[ -z "$PIPELINE_IP" ]]; then
  echo "✗ Could not determine pipeline worker public IP — cannot set CI_ALLOWED_CIDR"
  exit 1
fi
export TF_VAR_CI_ALLOWED_CIDR="${PIPELINE_IP}/32"
echo "✓ CI inbound rules locked to pipeline worker IP: $TF_VAR_CI_ALLOWED_CIDR"

# Remove stale Terraform state from previous pipeline runs to prevent checksum mismatches
rm -rf .terraform .terraform.lock.hcl
terraform init -no-color
terraform plan -no-color -out=tfplan

# Destroy on apply failure to prevent orphaned resources hitting account quotas
apply_failed=0
terraform apply -auto-approve -no-color tfplan || apply_failed=1

if [[ "$apply_failed" -eq 1 ]]; then
  echo "✗ terraform apply failed — running destroy to clean up partial resources"
  terraform destroy -auto-approve -no-color || true
  exit 1
fi

# Save outputs to WORKSPACE so acceptance-test stage (separate container) can access them
OUTPUT_FILE="${TF_OUTPUTS_PATH:-$TERRAFORM_DIR/terraform-outputs.json}"
terraform output -json > "$OUTPUT_FILE"
echo "✓ Deployment complete - outputs saved to $OUTPUT_FILE"

# Save state to WORKSPACE so destroy stage (separate container) can find the resources to remove
cp terraform.tfstate "$WORKSPACE/terraform.tfstate"
echo "✓ State saved to $WORKSPACE/terraform.tfstate"

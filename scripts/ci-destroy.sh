#!/usr/bin/env bash
# CI Pipeline - Destroy Script for FortiGate Standalone
# Sourced by terraform-ci-acceptance-test.sh after tests complete (pass or fail)
# TERRAFORM_DIR is set by the pipeline before sourcing this script

set -euo pipefail

echo "============================================"
echo "FortiGate Standalone - Destroy"
echo "============================================"

cd "$TERRAFORM_DIR"

# Load all TF_VAR_* exports and IC_API_KEY from the shared env loader.
# This script runs in a separate container from the deploy stage so env vars
# are not inherited — ci-env.sh re-reads them from the pipeline env store.
# shellcheck source=scripts/ci-env.sh
source "${TERRAFORM_DIR}/scripts/ci-env.sh"

# CI_MODE must match deploy to correctly manage state
export TF_VAR_CI_MODE=true

# Re-initialise providers — wipe stale cache first to prevent checksum mismatches
rm -rf .terraform .terraform.lock.hcl
terraform init -no-color

# Restore state from WORKSPACE so destroy knows which resources to remove
if [[ -f "$WORKSPACE/terraform.tfstate" ]]; then
  cp "$WORKSPACE/terraform.tfstate" terraform.tfstate
  echo "✓ State restored from $WORKSPACE/terraform.tfstate"
else
  echo "⚠ No state file found in WORKSPACE — destroy may not remove all resources"
fi

terraform destroy -auto-approve -no-color
echo "✓ Resources destroyed"

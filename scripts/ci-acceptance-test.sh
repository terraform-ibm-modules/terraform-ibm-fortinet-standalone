#!/usr/bin/env bash
# CI Pipeline - Acceptance Test Script for FortiGate Standalone 
# Sourced by terraform-ci-acceptance-test.sh in the acceptance-test stage
# TERRAFORM_DIR and TF_OUTPUTS_PATH are set by the pipeline before sourcing

set -euo pipefail

echo "============================================"
echo "FortiGate Standalone - Acceptance Tests"
echo "============================================"

cd "$TERRAFORM_DIR"

# Verify outputs file exists from deploy stage
TF_OUTPUTS_PATH="${TF_OUTPUTS_PATH:-$TERRAFORM_DIR/terraform-outputs.json}"
if [ ! -f "$TF_OUTPUTS_PATH" ]; then
  echo "✗ Terraform outputs not found at $TF_OUTPUTS_PATH"
  echo "  Deploy stage may have failed — check deploy stage logs"
  exit 1
fi
echo "✓ Terraform outputs found at $TF_OUTPUTS_PATH"

# CI_MODE=true enables acceptance tests in acceptance_test.go
export CI_MODE=true
export TF_OUTPUTS_PATH

cd "$TERRAFORM_DIR/tests"
echo "Running acceptance tests from: $(pwd)"

go test -v -timeout 30m -run "TestFortiGate|TestCatalogOffering"

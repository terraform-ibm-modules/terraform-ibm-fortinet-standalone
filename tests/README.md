# Terraform Validation Tests

This directory contains comprehensive validation tests for the FortiGate Standalone Terraform deployment.

## Test Suite Overview

The test suite includes **14 tests** covering basic validation, security, and resource-specific checks:

| # | Test Name | Category | Description | What It Validates |
|---|-----------|----------|-------------|-------------------|
| 1 | `TestTerraformValidation` | Basic | Runs `terraform init` and `terraform validate` | Terraform code is syntactically valid |
| 2 | `TestTerraformFormat` | Basic | Runs `terraform fmt -check -diff` | Code follows consistent formatting |
| 3 | `TestTerraformSyntax` | Basic | Checks for .tf files and basic structure | Project structure is valid |
| 4 | `TestTerraformVariables` | Basic | Checks for variables.tf | Variable definitions exist |
| 5 | `TestTerraformOutputs` | Basic | Checks for outputs.tf | Output definitions exist |
| 6 | `TestTerraformProviders` | Basic | Checks for provider configuration | Provider setup is valid |
| 7 | `TestNoHardcodedSecrets` | Security | Scans for hardcoded sensitive values | No passwords, API keys, secrets, or tokens hardcoded |
| 8 | `TestSecurityGroupConfiguration` | Security | Validates security group rules | Security group exists with proper rules and direction |
| 9 | `TestCatalogOfferingCRNs` | Resource | Validates catalog offering configuration | Version CRN and Plan CRN are properly parameterized |
| 10 | `TestVSIProfileSelection` | Resource | Validates VSI profile selection logic | Profile mapping, region overrides, and selected_profile usage |
| 11 | `TestMetadataServiceEnabled` | Resource | Validates metadata service configuration | Metadata service is enabled (required for VNF licensing) |
| 12 | `TestResourceNaming` | Resource | Validates resource naming conventions | Resources use CLUSTER_NAME and random suffix pattern |
| 13 | `TestNetworkConfiguration` | Resource | Validates network interface setup | Dual NIC, IP spoofing disabled, security groups attached |
| 14 | `TestRequiredVariables` | Resource | Validates all required variables are defined | All 11 required variables are present |

## Running Tests

### Run All Tests
```bash
cd tests
go test -v -timeout 5m
```

### Run Specific Test Category
```bash
# Security tests only
go test -v -run "TestNoHardcodedSecrets|TestSecurityGroup"

# Resource validation tests only
go test -v -run "TestCatalogOffering|TestVSIProfile|TestMetadata|TestResourceNaming|TestNetworkConfiguration|TestRequiredVariables"

# Basic validation tests only
go test -v -run "TestTerraformValidation|TestTerraformFormat"
```

### Run Single Test
```bash
go test -v -run TestNoHardcodedSecrets
```

## Test Execution Time

- **Local execution**: ~3 seconds (all tests run in parallel)
- **Pipeline execution**: ~2-3 minutes (includes Go/Terraform installation)

## Test Architecture

### Key Features

1. **Parallel Execution**: All tests run in parallel using `t.Parallel()`
2. **Context-Aware**: Uses context with timeouts for Terraform operations
3. **Non-Destructive**: No actual resources are deployed
4. **Environment-Driven**: Configurable via environment variables
5. **Fast Feedback**: Quick validation suitable for PR pipelines

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TERRAFORM_DIR` | Path to Terraform code | `../` (parent directory) |
| `TF_VAR_ibmcloud_api_key` | IBM Cloud API key | `dummy-api-key-for-validation` |

## Security Test Details

### TestNoHardcodedSecrets

Scans all `.tf` files for patterns indicating hardcoded secrets:

- **Passwords**: `password = "value"`
- **API Keys**: `api_key = "value"` or `api-key = "value"`
- **Secrets**: `secret = "value"`
- **Tokens**: `token = "value"`

**Note**: Uses regex patterns to detect values longer than minimum thresholds (3 chars for passwords, 10 chars for keys/secrets/tokens).

### TestSecurityGroupConfiguration

Validates security group configuration in `vpc.tf`:

- ✅ Security group resource exists
- ✅ Security group rules are defined
- ✅ Rule direction is specified
- ✅ No overly permissive inbound rules by default

## Resource Validation Details

### TestCatalogOfferingCRNs

Validates catalog offering integration in `instance.tf`:

- ✅ `catalog_offering` block exists
- ✅ `version_crn` is specified
- ✅ `plan_crn` is specified
- ✅ CRNs use variables (not hardcoded)

### TestVSIProfileSelection

Validates VSI profile selection logic in `locals.tf`:

- ✅ `plan_crn_to_profile` mapping exists
- ✅ `region_specific_profile_overrides` exists
- ✅ `selected_profile` logic is defined
- ✅ Instance uses `local.selected_profile`

### TestMetadataServiceEnabled

Validates metadata service configuration in `instance.tf`:

- ✅ `metadata_service` block exists
- ✅ `enabled` parameter is specified

**Why This Matters**: Metadata service is required for VNF licensing to work properly.

### TestResourceNaming

Validates resource naming conventions:

- ✅ `CLUSTER_NAME` variable is defined
- ✅ `random_string` resource exists for unique suffixes
- ✅ Resources use `var.CLUSTER_NAME`
- ✅ Resources use `random_string.random_suffix`

**Naming Pattern**: `${var.CLUSTER_NAME}-resource-${random_string.random_suffix.result}`

### TestNetworkConfiguration

Validates network interface setup in `vpc.tf`:

- ✅ Virtual network interfaces are defined
- ✅ `allow_ip_spoofing` is configured (should be false)
- ✅ Security groups are attached to interfaces
- ✅ Dual NIC configuration (primary + secondary)

### TestRequiredVariables

Validates all required variables are defined in `vars.tf`:

1. `IBMCLOUD_API_KEY`
2. `SSH_PUBLIC_KEY`
3. `REGION`
4. `ZONE1`
5. `VPC`
6. `SUBNET1`
7. `SUBNET2`
8. `CATALOG_OFFERING_VERSION_CRN`
9. `CATALOG_OFFERING_PLAN_CRN`
10. `CLUSTER_NAME`
11. `RESOURCE_GROUP`

## Integration with Pipeline

These tests are automatically run by the IBM OnePipeline PR pipeline:

1. **Setup Stage**: Installs Go, Terraform, and dependencies
2. **Test Stage**: Runs all tests in parallel
3. **Compliance Stage**: Runs detect-secrets and Mend scanning
4. **Finish Stage**: Reports results

## Pre-commit Hooks

Tests are also run locally via pre-commit hooks before each commit:

```bash
# Install pre-commit hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

The pre-commit hook runs:
- `TestTerraformFormat`
- `TestTerraformValidation`

## Adding New Tests

To add a new test:

1. Create a new test function in `pr_test.go`
2. Use `t.Parallel()` for parallel execution
3. Follow the naming convention: `TestXxx`
4. Add descriptive logging with `t.Log()` and `t.Logf()`
5. Use assertions from `github.com/stretchr/testify/assert`
6. Test locally before committing

Example:
```go
func TestMyNewValidation(t *testing.T) {
    t.Parallel()
    
    terraformDir := getTerraformDir()
    t.Log("Running my new validation...")
    
    // Your validation logic here
    
    t.Log("✓ Validation passed")
}
```

## Troubleshooting

### Tests Fail Locally

1. **Check Go version**: Requires Go 1.21+
2. **Check Terraform version**: Requires Terraform 1.6+
3. **Run from tests/ directory**: `cd tests && go test -v`
4. **Check environment variables**: Ensure `TERRAFORM_DIR` is set correctly

### Tests Pass Locally But Fail in Pipeline

1. **Check pipeline logs**: Look for setup errors
2. **Verify Go/Terraform versions**: Pipeline uses specific versions
3. **Check file paths**: Pipeline runs from different directory
4. **Review timeout settings**: Pipeline may need longer timeouts

## Best Practices

1. **Keep tests fast**: Aim for < 5 seconds total execution time
2. **Use parallel execution**: All tests should use `t.Parallel()`
3. **Avoid external dependencies**: Tests should be self-contained
4. **Use descriptive names**: Test names should clearly indicate what they validate
5. **Add logging**: Use `t.Log()` to provide feedback during test execution
6. **Handle errors gracefully**: Use `t.Skip()` for missing files, not `t.Fatal()`

## References

- [Terratest Documentation](https://terratest.gruntwork.io/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [IBM OnePipeline Documentation](https://cloud.ibm.com/docs/devsecops)
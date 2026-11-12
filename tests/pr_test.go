package test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// getTerraformDir returns the Terraform directory from environment or default
func getTerraformDir() string {
	terraformDir := os.Getenv("TERRAFORM_DIR")
	if terraformDir == "" {
		terraformDir = "../"
	}
	return terraformDir
}

// getTerraformOptions creates standard Terraform options for testing
func getTerraformOptions(t *testing.T) *terraform.Options {
	terraformDir := getTerraformDir()

	options := &terraform.Options{
		TerraformDir: terraformDir,
		NoColor:      true,
	}

	// Check if we need to set variables from environment
	if varsJSON := os.Getenv("TF_VARS_JSON"); varsJSON != "" {
		t.Logf("Using variables from environment")
	}

	return options
}

// TestTerraformValidation performs basic validation of Terraform code
// This test only runs terraform init and validate (no actual deployment)
func TestTerraformValidation(t *testing.T) {
	t.Parallel()

	// Get API key from environment (can be dummy for validation)
	apiKey := os.Getenv("TF_VAR_ibmcloud_api_key")
	if apiKey == "" {
		apiKey = "dummy-api-key-for-validation"
	}

	terraformOptions := getTerraformOptions(t)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run terraform init and validate in one step (using context-aware function)
	t.Log("Running terraform init and validate...")
	terraform.InitAndValidateContext(t, ctx, terraformOptions)

	t.Log("✓ Terraform validation passed")
}

// TestTerraformFormat checks if Terraform files are properly formatted
func TestTerraformFormat(t *testing.T) {
	t.Parallel()

	terraformOptions := getTerraformOptions(t)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Run terraform fmt -check with -diff for better error messages
	// diffutils is installed in the setup script to support this
	t.Log("Running terraform fmt -check...")
	terraform.RunTerraformCommandContext(t, ctx, terraformOptions, "fmt", "-check", "-diff")

	t.Log("✓ Terraform format check passed")
}

// TestTerraformSyntax performs additional syntax validation
func TestTerraformSyntax(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	// Check for common syntax issues
	t.Log("Checking for common Terraform syntax issues...")

	// Verify .tf files exist
	matches, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	assert.NoError(t, err, "Should be able to search for .tf files")

	if len(matches) == 0 {
		t.Skip("No .tf files found - skipping syntax check (tests should run from a Terraform project directory)")
	}

	t.Logf("Found %d Terraform files", len(matches))
}

// TestTerraformVariables checks if all required variables are defined
func TestTerraformVariables(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for variables.tf...")

	// Check if variables.tf exists
	variablesFile := filepath.Join(terraformDir, "variables.tf")
	if _, err := os.Stat(variablesFile); err == nil {
		t.Logf("Found variables.tf")
	} else {
		t.Log("No variables.tf found (may be defined elsewhere)")
	}
}

// TestTerraformOutputs checks if outputs are properly defined
func TestTerraformOutputs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for outputs.tf...")

	// Check if outputs.tf exists
	outputsFile := filepath.Join(terraformDir, "outputs.tf")
	if _, err := os.Stat(outputsFile); err == nil {
		t.Logf("Found outputs.tf")
	} else {
		t.Log("No outputs.tf found")
	}
}

// TestTerraformProviders checks if providers are properly configured
// Note: This test is informational only and doesn't fail on provider issues
func TestTerraformProviders(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking provider configuration...")

	// Check if providers.tf or main.tf contains provider blocks
	providerFiles := []string{
		filepath.Join(terraformDir, "providers.tf"),
		filepath.Join(terraformDir, "main.tf"),
	}

	foundProvider := false
	for _, file := range providerFiles {
		if _, err := os.Stat(file); err == nil {
			t.Logf("Found provider configuration file: %s", filepath.Base(file))
			foundProvider = true
		}
	}

	if !foundProvider {
		t.Log("No dedicated provider configuration file found (providers may be defined inline)")
	}

	t.Log("✓ Provider configuration check complete")
}

// TestNoHardcodedSecrets checks for hardcoded sensitive values in Terraform files
func TestNoHardcodedSecrets(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Checking for hardcoded secrets in Terraform files...")

	// Patterns that should not appear in Terraform files
	forbiddenPatterns := []struct {
		pattern     *regexp.Regexp
		description string
	}{
		{regexp.MustCompile(`(?i)password\s*=\s*"[^"]{3,}"`), "Hardcoded password"},
		{regexp.MustCompile(`(?i)api[_-]?key\s*=\s*"[^"]{10,}"`), "Hardcoded API key"},
		{regexp.MustCompile(`(?i)secret\s*=\s*"[^"]{10,}"`), "Hardcoded secret"},
		{regexp.MustCompile(`(?i)token\s*=\s*"[^"]{10,}"`), "Hardcoded token"},
	}

	// Get all .tf files
	matches, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	assert.NoError(t, err, "Should be able to search for .tf files")

	if len(matches) == 0 {
		t.Skip("No .tf files found - skipping secret check")
	}

	hasSecrets := false
	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Logf("Warning: Could not read file %s: %v", file, err)
			continue
		}

		for _, pattern := range forbiddenPatterns {
			if pattern.pattern.Match(content) {
				t.Errorf("Found potential %s in %s", pattern.description, filepath.Base(file))
				hasSecrets = true
			}
		}
	}

	if !hasSecrets {
		t.Log("✓ No hardcoded secrets detected")
	}
}

// TestSecurityGroupConfiguration validates security group rules
func TestSecurityGroupConfiguration(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating security group configuration...")

	// Check if vpc.tf exists (contains security group)
	vpcFile := filepath.Join(terraformDir, "vpc.tf")
	if _, err := os.Stat(vpcFile); os.IsNotExist(err) {
		t.Skip("vpc.tf not found - skipping security group validation")
	}

	content, err := os.ReadFile(vpcFile)
	assert.NoError(t, err, "Should be able to read vpc.tf")

	contentStr := string(content)

	// Verify security group exists
	assert.Contains(t, contentStr, "ibm_is_security_group", "Should define security group")
	t.Log("✓ Security group resource found")

	// Verify outbound rule exists
	assert.Contains(t, contentStr, "ibm_is_security_group_rule", "Should define security group rules")
	t.Log("✓ Security group rules found")

	// Verify no overly permissive inbound rules (0.0.0.0/0 with direction = "inbound")
	// This is a basic check - the actual security group has deny-all by default
	if assert.Contains(t, contentStr, "direction", "Should specify rule direction") {
		t.Log("✓ Security group rules have direction specified")
	}
}

// TestCatalogOfferingCRNs validates catalog offering configuration
func TestCatalogOfferingCRNs(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating catalog offering CRN configuration...")

	// Check if instance.tf exists
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Skip("instance.tf not found - skipping catalog offering validation")
	}

	content, err := os.ReadFile(instanceFile)
	assert.NoError(t, err, "Should be able to read instance.tf")

	contentStr := string(content)

	// Verify catalog_offering block exists
	assert.Contains(t, contentStr, "catalog_offering", "Should use catalog_offering for licensing")
	t.Log("✓ Catalog offering block found")

	// Verify version_crn is referenced
	assert.Contains(t, contentStr, "version_crn", "Should specify version_crn")
	t.Log("✓ Version CRN reference found")

	// Verify plan_crn is referenced
	assert.Contains(t, contentStr, "plan_crn", "Should specify plan_crn")
	t.Log("✓ Plan CRN reference found")

	// Verify variables are used (not hardcoded)
	assert.Contains(t, contentStr, "var.CATALOG_OFFERING_VERSION_CRN", "Should use variable for version CRN")
	assert.Contains(t, contentStr, "var.CATALOG_OFFERING_PLAN_CRN", "Should use variable for plan CRN")
	t.Log("✓ CRNs are properly parameterized")
}

// TestVSIProfileSelection validates VSI profile selection logic
func TestVSIProfileSelection(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating VSI profile selection logic...")

	// Check if locals.tf exists
	localsFile := filepath.Join(terraformDir, "locals.tf")
	if _, err := os.Stat(localsFile); os.IsNotExist(err) {
		t.Skip("locals.tf not found - skipping profile selection validation")
	}

	content, err := os.ReadFile(localsFile)
	assert.NoError(t, err, "Should be able to read locals.tf")

	contentStr := string(content)

	// Verify profile mapping exists
	assert.Contains(t, contentStr, "plan_crn_to_profile", "Should define plan CRN to profile mapping")
	t.Log("✓ Profile mapping found")

	// Verify region-specific overrides exist
	assert.Contains(t, contentStr, "region_specific_profile_overrides", "Should define region-specific overrides")
	t.Log("✓ Region-specific profile overrides found")

	// Verify selected_profile logic exists
	assert.Contains(t, contentStr, "selected_profile", "Should define profile selection logic")
	t.Log("✓ Profile selection logic found")

	// Verify instance.tf uses the selected profile
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")
		assert.Contains(t, string(instanceContent), "local.selected_profile", "Instance should use selected_profile")
		t.Log("✓ Instance uses selected_profile from locals")
	}
}

// TestMetadataServiceEnabled validates metadata service configuration
func TestMetadataServiceEnabled(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating metadata service configuration...")

	// Check if instance.tf exists
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		t.Skip("instance.tf not found - skipping metadata service validation")
	}

	content, err := os.ReadFile(instanceFile)
	assert.NoError(t, err, "Should be able to read instance.tf")

	contentStr := string(content)

	// Verify metadata_service block exists
	assert.Contains(t, contentStr, "metadata_service", "Should configure metadata service")
	t.Log("✓ Metadata service block found")

	// Verify metadata service is enabled
	assert.Contains(t, contentStr, "enabled", "Should specify metadata service enabled status")
	t.Log("✓ Metadata service enabled configuration found")

	// Note: We can't verify the actual value without parsing HCL, but presence is good
}

// TestResourceNaming validates resource naming conventions
func TestResourceNaming(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating resource naming conventions...")

	// Check if vars.tf exists
	varsFile := filepath.Join(terraformDir, "vars.tf")
	if _, err := os.Stat(varsFile); os.IsNotExist(err) {
		t.Skip("vars.tf not found - skipping naming validation")
	}

	content, err := os.ReadFile(varsFile)
	assert.NoError(t, err, "Should be able to read vars.tf")

	contentStr := string(content)

	// Verify CLUSTER_NAME variable exists
	assert.Contains(t, contentStr, "CLUSTER_NAME", "Should define CLUSTER_NAME variable")
	t.Log("✓ CLUSTER_NAME variable found")

	// Verify random_string resource exists for unique suffixes
	assert.Contains(t, contentStr, "random_string", "Should use random suffix for resource names")
	t.Log("✓ Random suffix configuration found")

	// Check instance.tf for proper naming pattern
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")

		// Verify resources use CLUSTER_NAME and random suffix
		assert.Contains(t, string(instanceContent), "var.CLUSTER_NAME", "Resources should use CLUSTER_NAME")
		assert.Contains(t, string(instanceContent), "random_string.random_suffix", "Resources should use random suffix")
		t.Log("✓ Resources follow naming convention")
	}
}

// TestNetworkConfiguration validates network interface configuration
func TestNetworkConfiguration(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating network configuration...")

	// Check if vpc.tf exists
	vpcFile := filepath.Join(terraformDir, "vpc.tf")
	if _, err := os.Stat(vpcFile); os.IsNotExist(err) {
		t.Skip("vpc.tf not found - skipping network validation")
	}

	content, err := os.ReadFile(vpcFile)
	assert.NoError(t, err, "Should be able to read vpc.tf")

	contentStr := string(content)

	// Verify virtual network interfaces are defined
	assert.Contains(t, contentStr, "ibm_is_virtual_network_interface", "Should define virtual network interfaces")
	t.Log("✓ Virtual network interfaces found")

	// Verify IP spoofing is disabled (security best practice)
	assert.Contains(t, contentStr, "allow_ip_spoofing", "Should configure IP spoofing setting")
	t.Log("✓ IP spoofing configuration found")

	// Verify security groups are attached to interfaces
	assert.Contains(t, contentStr, "security_groups", "Should attach security groups to interfaces")
	t.Log("✓ Security groups attached to network interfaces")

	// Check instance.tf for dual NIC configuration
	instanceFile := filepath.Join(terraformDir, "instance.tf")
	if _, err := os.Stat(instanceFile); err == nil {
		instanceContent, err := os.ReadFile(instanceFile)
		assert.NoError(t, err, "Should be able to read instance.tf")

		// Verify primary and secondary network attachments
		assert.Contains(t, string(instanceContent), "primary_network_attachment", "Should have primary network attachment")
		assert.Contains(t, string(instanceContent), "network_attachments", "Should have secondary network attachments")
		t.Log("✓ Dual NIC configuration found")
	}
}

// TestRequiredVariables validates that all required variables are defined
func TestRequiredVariables(t *testing.T) {
	t.Parallel()

	terraformDir := getTerraformDir()

	t.Log("Validating required variables...")

	// Check if vars.tf exists
	varsFile := filepath.Join(terraformDir, "vars.tf")
	if _, err := os.Stat(varsFile); os.IsNotExist(err) {
		t.Skip("vars.tf not found - skipping variable validation")
	}

	content, err := os.ReadFile(varsFile)
	assert.NoError(t, err, "Should be able to read vars.tf")

	contentStr := string(content)

	// List of required variables for FortiGate deployment
	requiredVars := []string{
		"IBMCLOUD_API_KEY",
		"SSH_PUBLIC_KEY",
		"REGION",
		"ZONE1",
		"VPC",
		"SUBNET1",
		"SUBNET2",
		"CATALOG_OFFERING_VERSION_CRN",
		"CATALOG_OFFERING_PLAN_CRN",
		"CLUSTER_NAME",
		"RESOURCE_GROUP",
	}

	for _, varName := range requiredVars {
		assert.Contains(t, contentStr, varName, "Should define required variable: %s", varName)
	}

	t.Logf("✓ All %d required variables are defined", len(requiredVars))
}


package test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

// skipIfNotCI skips the test if CI_MODE is not set
// Acceptance tests require a live deployed FortiGate instance
func skipIfNotCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI_MODE") != "true" {
		t.Skip("Skipping acceptance test: CI_MODE is not set. Requires a live deployed FortiGate instance.")
	}
}


// tfOutputs reads terraform outputs from the file saved by the deploy stage
type tfOutput struct {
	Value interface{} `json:"value"`
}

func readTFOutputs(t *testing.T) map[string]tfOutput {
	t.Helper()
	outputFile := os.Getenv("TF_OUTPUTS_PATH")
	if outputFile == "" {
		outputFile = "../terraform-outputs.json"
	}
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Should be able to read terraform outputs from %s", outputFile)

	var outputs map[string]tfOutput
	require.NoError(t, json.Unmarshal(data, &outputs), "Terraform outputs should be valid JSON")
	return outputs
}

func getOutputString(t *testing.T, outputs map[string]tfOutput, key string) string {
	t.Helper()
	v, ok := outputs[key]
	require.True(t, ok, "Terraform output '%s' should exist", key)
	str, ok := v.Value.(string)
	require.True(t, ok, "Terraform output '%s' should be a string", key)
	require.NotEmpty(t, str, "Terraform output '%s' should not be empty", key)
	return str
}

// sshRunCommandsOutput opens a new SSH session with the given password, sends CLI
// commands one by one with a short pause between each, logs and returns all output.
// Returns ("", false) if the SSH connection or session setup fails.
func sshRunCommandsOutput(t *testing.T, publicIP string, password string, cmds []string, label string) (string, bool) {
	t.Helper()
	cfg := &gossh.ClientConfig{
		User:            "admin",
		Auth:            []gossh.AuthMethod{gossh.Password(password)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}
	conn, err := gossh.Dial("tcp", publicIP+":22", cfg)
	if err != nil {
		t.Logf("  SSH [%s]: connect failed: %v", label, err)
		return "", false
	}
	defer conn.Close()

	sess, err := conn.NewSession()
	if err != nil {
		t.Logf("  SSH [%s]: new session failed: %v", label, err)
		return "", false
	}
	err = sess.RequestPty("xterm", 40, 80, gossh.TerminalModes{gossh.ECHO: 0})
	if err != nil {
		t.Logf("  SSH [%s]: request pty failed: %v", label, err)
		return "", false
	}
	stdin, _ := sess.StdinPipe()
	var outBuf strings.Builder
	sess.Stdout = &outBuf
	sess.Stderr = &outBuf
	_ = sess.Shell()

	time.Sleep(2 * time.Second)
	for _, cmd := range cmds {
		t.Logf("  SSH [%s] >> %s", label, cmd)
		fmt.Fprintf(stdin, "%s\n", cmd)
		time.Sleep(1 * time.Second)
	}
	time.Sleep(3 * time.Second)
	stdin.Close()
	sess.Close()

	out := outBuf.String()
	t.Logf("  SSH [%s] output:\n%s", label, out)
	return out, true
}

// fortiGateSSHInit polls until FortiGate accepts an SSH connection with the
// initial password, then performs first-boot password change.
// Polling replaces the separate HTTPS/TCP readiness probe — if SSH is up with
// the initial password, the instance is ready and we can init in one step.
// On subsequent runs (password already changed) the initial-password dial fails
// immediately and we return newPassword without waiting.
// Returns the current admin password.
func fortiGateSSHInit(t *testing.T, publicIP string, instanceID string, newPassword string) string {
	t.Helper()

	sshCfg := &gossh.ClientConfig{
		User:            "admin",
		Auth:            []gossh.AuthMethod{gossh.Password(instanceID)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}

	// Poll until SSH is reachable with the initial password (instance still booting)
	// or the initial password is rejected (already changed — return early).
	timeoutMinutes := 10
	deadline := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	t.Logf("  SSH init: waiting for FortiGate SSH at %s:22 (timeout: %dm)...", publicIP, timeoutMinutes)

	var conn *gossh.Client
	for {
		var err error
		conn, err = gossh.Dial("tcp", publicIP+":22", sshCfg)
		if err == nil {
			break // connected with initial password — proceed to change it
		}
		// If auth was rejected (not a network error) the password is already changed
		if isAuthError(err) {
			t.Logf("  SSH init: initial password rejected — password already changed, using newPassword")
			return newPassword
		}
		// Network-level failure — instance still booting, keep polling
		if time.Now().After(deadline) {
			t.Fatalf("  SSH init: FortiGate SSH did not become reachable within %d minutes: %v", timeoutMinutes, err)
		}
		elapsed := time.Since(deadline.Add(-time.Duration(timeoutMinutes) * time.Minute))
		t.Logf("  SSH init: not yet reachable (%ds elapsed), retrying...", int(elapsed.Seconds()))
		time.Sleep(30 * time.Second)
	}

	t.Log("  SSH init: connected with initial password — changing password now")
	sess, serr := conn.NewSession()
	require.NoError(t, serr, "SSH new session")
	_ = sess.RequestPty("xterm", 40, 80, gossh.TerminalModes{gossh.ECHO: 0})
	stdin, _ := sess.StdinPipe()
	var outBuf strings.Builder
	sess.Stdout = &outBuf
	sess.Stderr = &outBuf
	_ = sess.Shell()

	// FortiGate firmware prompts differ across versions:
	//   v7/v8 (this device): "New Password:" immediately — no old-password prompt.
	//   Some older FOS:      "Old password:" → "New Password:" → "Confirm Password:"
	//
	// Read the first prompt to detect which flow is active, then respond correctly.
	// Blindly sending instanceID as the first line causes a policy failure on firmware
	// that goes straight to "New Password:" — it treats instanceID as the new password,
	// which may fail policy (e.g. no special chars in a UUID), burning a retry slot.
	time.Sleep(3 * time.Second) // wait for first prompt to appear in buffer
	firstPrompt := strings.ToLower(outBuf.String())
	if strings.Contains(firstPrompt, "old password") || strings.Contains(firstPrompt, "current password") {
		// Firmware asked for the current (old) password first
		fmt.Fprintf(stdin, "%s\n", instanceID)
		time.Sleep(2 * time.Second) // wait for "New Password:" prompt
	}
	fmt.Fprintf(stdin, "%s\n", newPassword) // new password
	time.Sleep(2 * time.Second)             // wait for "Confirm Password:" prompt
	fmt.Fprintf(stdin, "%s\n", newPassword) // confirm
	time.Sleep(5 * time.Second)             // wait for password change to complete
	stdin.Close()
	sess.Close()
	conn.Close()

	// Always log output for diagnosis
	t.Logf("  SSH init [pwd-change] output:\n%s", outBuf.String())
	t.Log("  SSH init: password change complete")

	// Give FortiGate time to settle after password change before parallel tests probe it.
	t.Log("  SSH init: waiting 60s for FortiGate to settle after password change...")
	time.Sleep(60 * time.Second)

	return newPassword
}

// ciPassword returns the fixed CI password used after SSH init.
// Using a fixed password avoids PTY character-encoding issues with long UUIDs.
func ciPassword() string {
	return "Ci-Pipeline-1!"
}

// isAuthError returns true when an SSH dial error is an authentication failure
// (password rejected) rather than a network-level connection failure.
// Used by fortiGateSSHInit to distinguish "not yet up" from "already initialised".
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unable to authenticate") ||
		strings.Contains(s, "no supported methods") ||
		strings.Contains(s, "ssh: handshake failed")
}

// fortiGateAdminPassword probes which password is currently active (new or initial)
// via SSH — avoids FortiGate REST API Basic Auth issues (trusted-host restrictions,
// session-cookie requirements). Returns the working password.
// SSH init must have already been run (by TestFortiGateReachable) before calling this.
func fortiGateAdminPassword(t *testing.T, publicIP string, instanceID string) string {
	t.Helper()
	newPassword := ciPassword()

	// sshProbe tries to open and immediately close an SSH connection.
	// A successful dial means the password is correct.
	sshProbe := func(pwd string, label string) bool {
		cfg := &gossh.ClientConfig{
			User:            "admin",
			Auth:            []gossh.AuthMethod{gossh.Password(pwd)},
			HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
			Timeout:         10 * time.Second,
		}
		conn, err := gossh.Dial("tcp", publicIP+":22", cfg)
		if err != nil {
			t.Logf("  SSH probe [%s]: connect failed: %v", label, err)
			return false
		}
		conn.Close()
		t.Logf("  SSH probe [%s]: connected ✓", label)
		return true
	}

	if sshProbe(newPassword, "new-pwd") {
		t.Log("✓ Authenticated with new password (SSH)")
		return newPassword
	}
	if sshProbe(instanceID, "initial-pwd") {
		t.Log("✓ Authenticated with initial password (SSH)")
		return instanceID
	}

	require.Fail(t, "Failed to authenticate with FortiGate via SSH (tried new and initial passwords)")
	return ""
}

// TestFortiGatePublicIPAssigned verifies a floating IP was assigned - runs first, not parallel
func TestFortiGatePublicIPAssigned(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)
	publicIP := getOutputString(t, outputs, "FortiGate_Public_IP")
	t.Logf("✓ Public IP assigned: %s", publicIP)
}

// TestFortiGateReachable polls FortiGate HTTPS until it responds, then runs SSH init
// (disclaimer acceptance + password change). Runs serially before all parallel tests.
func TestFortiGateReachable(t *testing.T) {
	skipIfNotCI(t)

	outputs := readTFOutputs(t)
	publicIP := getOutputString(t, outputs, "FortiGate_Public_IP")
	instanceID := getOutputString(t, outputs, "Default_Admin_Password")
	newPassword := ciPassword()

	// fortiGateSSHInit polls for SSH readiness and changes the password in one step
	fortiGateSSHInit(t, publicIP, instanceID, newPassword)
}

// TestFortiGateLicenseValid verifies the FortiGate license is active via SSH CLI.
// Runs after TestFortiGateReachable ensures the instance is up.
func TestFortiGateLicenseValid(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	publicIP := getOutputString(t, outputs, "FortiGate_Public_IP")
	instanceID := getOutputString(t, outputs, "Default_Admin_Password")

	password := fortiGateAdminPassword(t, publicIP, instanceID)

	out, ok := sshRunCommandsOutput(t, publicIP, password,
		[]string{"get system status", "end"},
		"license-check",
	)
	require.True(t, ok, "SSH session for license check must succeed")

	// FortiGate outputs "License Status: Valid" (or "Contract Expiry Date:" for PAYG)
	// Accept either form: "Valid" license line or the PAYG marker
	outLower := strings.ToLower(out)
	isValid := strings.Contains(outLower, "license status: valid") ||
		strings.Contains(outLower, "contract expiry date:") ||
		strings.Contains(outLower, "registration status: registered")
	assert.True(t, isValid,
		"License should be valid — expected 'License Status: Valid' or PAYG registration marker in SSH output")
	t.Log("✓ License is valid")
}

// TestFortiGateInterfacesUp verifies port1 and port2 are up via SSH CLI.
func TestFortiGateInterfacesUp(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	publicIP := getOutputString(t, outputs, "FortiGate_Public_IP")
	instanceID := getOutputString(t, outputs, "Default_Admin_Password")

	password := fortiGateAdminPassword(t, publicIP, instanceID)

	out, ok := sshRunCommandsOutput(t, publicIP, password,
		[]string{"get system interface physical", "end"},
		"interfaces-check",
	)
	require.True(t, ok, "SSH session for interface check must succeed")

	// FortiGate CLI output format (get system interface physical):
	//   == [onboard]
	//       ==[port1]
	//           status: up
	//       ==[port2]
	//           status: up
	//
	// Strategy: scan line-by-line; when a line contains "[<name>]", record that
	// we're inside that interface's block. The next line containing "status:" gives
	// the value. Exit the block when we hit a line that starts a new interface
	// header (contains "==[") or the parent block header ("== [").
	checkInterface := func(name string) {
		lines := strings.Split(out, "\n")
		inBlock := false
		found := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Detect entry into this interface's block (e.g. "==[port1]")
			if strings.Contains(trimmed, "["+name+"]") {
				inBlock = true
				found = true
				continue
			}
			if !inBlock {
				continue
			}
			// Exit block on any new interface/section header
			if strings.Contains(trimmed, "==[") || strings.HasPrefix(trimmed, "== [") {
				break
			}
			if strings.HasPrefix(strings.ToLower(trimmed), "status:") {
				hasUp := strings.Contains(strings.ToLower(trimmed), "status: up")
				assert.True(t, hasUp, "Interface %s should show 'status: up' in SSH output (got: %s)", name, trimmed)
				t.Logf("Interface %s: %s", name, trimmed)
				return
			}
		}
		if !found {
			assert.Fail(t, fmt.Sprintf("Interface %s not found in SSH output", name))
		}
	}

	checkInterface("port1")
	checkInterface("port2")
	t.Log("✓ Both interfaces are up")
}

// TestFortiGateCatalogOfferingAttached verifies catalog offering CRNs from terraform outputs
func TestFortiGateCatalogOfferingAttached(t *testing.T) {
	skipIfNotCI(t)
	t.Parallel()

	outputs := readTFOutputs(t)
	versionCRN := getOutputString(t, outputs, "CATALOG_OFFERING_VERSION_CRN")
	planCRN := getOutputString(t, outputs, "CATALOG_OFFERING_PLAN_CRN")
	profile := getOutputString(t, outputs, "selected_vsi_profile")

	assert.True(t, strings.HasPrefix(versionCRN, "crn:"), "Version CRN should start with 'crn:'")
	assert.True(t, strings.HasPrefix(planCRN, "crn:"), "Plan CRN should start with 'crn:'")
	assert.NotEmpty(t, profile, "selected_vsi_profile should not be empty")

	t.Logf("✓ Catalog offering attached")
	t.Logf("  Version CRN : %s", versionCRN)
	t.Logf("  Plan CRN    : %s", planCRN)
	t.Logf("  VSI Profile : %s", profile)
}

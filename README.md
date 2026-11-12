# FortiGate Standalone Deployment on IBM Cloud

## Description

A Terraform based script that deploys a single FortiGate instance. This script is used by the IBM Cloud Catalog offering `Fortinet FortiGate VM NGFW - Single`. This template provides a simple, single-instance deployment suitable for development, testing, or small-scale production environments. 

- This terraform will deploy a FortiGate license for the selected pricing plan.
- Users may submit support tickets to IBM Cloud for issues relating to this offering.
- The Virtual FortiGate is customer-managed. Details on roles and responsibilities can be found [here](https://cloud.ibm.com/docs/licensed-firewall?topic=licensed-firewall-fortinet-racii-v3)

## Post-Deployment Considerations

### Firmware and Vulnerability Patching

For complete information about security maintenance and vulnerability management, see [Security maintenance and vulnerability management](https://cloud.ibm.com/docs/licensed-firewall?topic=licensed-firewall-security-maintenance-vulnerability-management).

Fortinet is responsible for identifying, investigating, and publishing security vulnerabilities that affect FortiGate products. Fortinet also provides security advisories, remediation guidance, firmware updates, and software fixes for affected products.

Your responsibilities for maintaining and operating your FortiGate deployment are defined in your service agreement and other applicable service documentation. Review your service agreement to understand your obligations regarding software maintenance, security updates, and vulnerability management.

**Monitoring security advisories**

Fortinet publishes security advisories through its Product Security Incident Response Team (PSIRT). Advisories include information about affected product versions, vulnerability severity, available mitigations, and recommended software updates.

IBM recommends that you regularly review Fortinet security advisories and evaluate whether remediation actions are required for your environment.

For more information, see the [Fortinet PSIRT portal](https://www.fortiguard.com/psirt).

**Subscribe to Fortinet notifications**

To receive notifications about security advisories, firmware releases, and other product updates, subscribe to Fortinet notification services.

Fortinet provides RSS feeds for security advisories, firmware releases, threat intelligence updates, and other product announcements. For more information, see [Manage Your Subscriptions to Fortinet](https://support.fortinet.com/).

**Accessing the FortiGate web management console**

Firmware upgrades require access to the FortiGate web management console on port 443. By default, this deployment attaches a Floating IP (FIP) to the FortiGate with deny-all inbound access for security. You can access the console using one of the following methods:

- **Floating IP with Allowlist**: Modify the security group rules to allowlist your source IP address for HTTPS (port 443) access to the attached FIP. This enables direct access to the FortiGate web console from the internet.
- **VPN Access (No FIP Required)**: Configure a VPN connection into your VPC to access the FortiGate management interface using its private IP address on port 443. This method does not require a Floating IP and provides secure private network access.

**Network requirements**

For firmware upgrades and license validation to function properly:

- **Egress Internet Access**: An egress route to the internet must be configured to allow the FortiGate to download firmware updates and validate the installed license.
- **Security Group Rules**: Ensure appropriate security group rules are in place to allow HTTPS (port 443) access from your management source IP addresses (when using FIP) or from your VPN subnet (when using VPN access).

**Best practices**

IBM recommends that you:

- Subscribe to Fortinet security notifications.
- Review Fortinet PSIRT advisories regularly.
- Maintain supported FortiOS versions.
- Apply security updates and remediation actions according to your organization's operational and security requirements.
- Include Fortinet security advisories in your vulnerability management processes.
- Ensure secure access to the FortiGate web management console before attempting firmware upgrades.
- Verify egress internet connectivity is available for firmware downloads and license validation.


## Requirements

-   [Terraform](https://learn.hashicorp.com/terraform/getting-started/install.html) 1.6+
-   IBM Cloud Terraform Provider version 2.1.0+
-   IBM Cloud Catalog offering CRNs (Version CRN and Plan CRN) for FortiGate
-   [A VPC with two subnets in a single zone](https://cloud.ibm.com/docs/vpc/vpc-getting-started-with-ibm-cloud-virtual-private-cloud-infrastructure)
-   [A configured IBM SSH key](https://cloud.ibm.com/docs/vpc?topic=vpc-ssh-keys)
-   IBM Cloud API key with appropriate permissions

## Key Features

- **Automatic Profile Selection**: VSI profile is automatically selected based on the catalog offering plan CRN
- **Region-Specific Support**: Handles region-specific profile availability (e.g., Chennai region where cx2 profiles are unavailable)
- **Dual Security Groups**: Two distinct security groups are automatically created — `sg-public` (port1) and `sg-private` (port2) — each with highly restrictive rules:
  - Inbound: deny all (no rules = implicit deny in IBM VPC)
  - Public SG outbound: restricted to FortiGuard services only (UDP 53, TCP 853, TCP 443, TCP 8890)
  - Private SG outbound: self-referencing only (no internet access on internal port)
- **Metadata Service Enabled**: Required for VNF licensing
- **Catalog Offering Support**: Uses IBM Cloud Catalog offerings instead of custom images

## Deployment overview

Terraform deploys the following components:

-   One FortiGate licensed instance (via IBM Cloud Catalog) with two NICs, one in each subnet
-   One floating Public IP address attached to the FortiGate on Port1
-   One log disk
-   Two security groups (`sg-public` for port1, `sg-private` for port2) with deny-all inbound and restricted outbound rules
-   A basic bootstrap configuration

## Configuration

### IBM Cloud Authentication
```hcl
IBMCLOUD_API_KEY = "your-ibm-cloud-api-key"
SSH_PUBLIC_KEY   = "your-ssh-key-name"
```

### Region and Zone
```hcl
REGION = "us-south"  # IBM Cloud region
ZONE1  = "us-south-1"  # Availability zone
```

### VPC and Network Configuration
```hcl
VPC     = "your-vpc-name"
SUBNET1 = "subnet-uuid-for-port1"  # Primary/Public subnet
SUBNET2 = "subnet-uuid-for-port2"  # Secondary/Private subnet
```

> **Note:** For Subnets, the UUID is required, not the name.

### Resource Configuration
```hcl
CLUSTER_NAME = "fortigate-terraform"  # Must be lowercase
RESOURCE_GROUP = "Default"
```

### Automatic Profile Selection

The VSI profile is automatically selected based on the `CATALOG_OFFERING_PLAN_CRN`. The mapping includes:

**Enterprise Plans:**
- 8vCPU Medium → cx3d-8x20
- 16vCPU Large → cx3d-16x40
- 32vCPU X-large → cx3d-32x80

**UTP Plans:**
- 2vCPU Small → cx3d-2x5
- 8vCPU Medium → cx3d-8x20
- 16vCPU Large → cx3d-16x40

**ATP Plans:**
- 2vCPU Small → cx2-2x4
- 8vCPU Medium → cx2-8x16

Region-specific overrides are automatically applied where certain profiles are unavailable (e.g., Chennai region).

## Outputs

The deployment provides the following outputs:

- `FortiGate_Public_IP`: Public IP address for the FortiGate instance
- `Security_Group_Public_ID`: ID of the public security group (port1)
- `Security_Group_Public_Name`: Name of the public security group (port1)
- `Security_Group_Private_ID`: ID of the private security group (port2)
- `Security_Group_Private_Name`: Name of the private security group (port2)
- `selected_vsi_profile`: The VSI profile automatically selected based on the plan CRN
- `CATALOG_OFFERING_VERSION_CRN`: The catalog offering version CRN used
- `CATALOG_OFFERING_PLAN_CRN`: The catalog offering plan CRN used
- `Username`: Default admin username (admin)
- `Default_Admin_Password`: Instance ID (password may be empty on first boot)

## Security Group Configuration

Two distinct security groups are automatically created, separating the public-facing and private interfaces:

### `sg-public` — port1 (external)

| Rule | Direction | Remote | Protocol | Port | Purpose |
|------|-----------|--------|----------|------|---------|
| *(none)* | Inbound | — | — | — | **Deny all** (implicit) |
| `allow-outbound-fortiguard-dns-udp-53` | Outbound | 0.0.0.0/0 | UDP | 53 | FortiGuard DNS & SDNS |
| `allow-outbound-dns-over-tls-tcp-853` | Outbound | 0.0.0.0/0 | TCP | 853 | DNS over TLS (DoT) |
| `allow-outbound-fortiguard-licensing-tcp-443` | Outbound | 0.0.0.0/0 | TCP | 443 | Licensing, FortiCare & Entitlements |
| `allow-outbound-fortiguard-updates-tcp-8890` | Outbound | 0.0.0.0/0 | TCP | 8890 | FortiGuard Distribution Updates |
| `allow-outbound-ntp-udp-123` | Outbound | 0.0.0.0/0 | UDP | 123 | NTP Time Synchronization |

### `sg-private` — port2 (internal)

| Rule | Direction | Remote | Protocol | Port | Purpose |
|------|-----------|--------|----------|------|---------|
| *(none)* | Inbound | — | — | — | **Deny all** (implicit) |
| `allow-outbound-to-same-sg` | Outbound | This SG | All | All | Internal traffic only |

### Security Benefits
- **Interface Segregation**: Public and private ports have separate security boundaries
- **Deny All Inbound**: No inbound rules means zero inbound access on both interfaces by default
- **Restricted Outbound**: Public SG only allows FortiGuard service ports (UDP 53, TCP 853, TCP 443, TCP 8890) — no broad `0.0.0.0/0` on all ports
- **Air-gapped Internal Port**: Private SG has zero internet access — internal traffic stays fully isolated
- **Named Rules**: All rules have descriptive names for easy identification in IBM Cloud console

### Customization
You can modify these rules in `vpc.tf` as needed. To allow management access to the FortiGate:
- Add an inbound rule to `sg-public` for HTTPS (port 443) from your management IP addresses
- Or use VPN access to reach the FortiGate management interface privately via its port1 private IP

**Important**: The FortiGate instance handles all traffic forwarding and security policies internally. The security groups only control access to the FortiGate management interface.

## Catalog Offering Integration

This deployment uses IBM Cloud Catalog offerings for licensing:

```hcl
catalog_offering {
  version_crn = var.CATALOG_OFFERING_VERSION_CRN
  plan_crn    = var.CATALOG_OFFERING_PLAN_CRN
}
```

The catalog offering CRNs are automatically configured when deploying from the IBM Cloud Catalog.

## Troubleshooting

### Common Issues

1. **Viewing Terraform Execution Logs**
   - The Firewall VSI installation creates a Schematics workspace that displays terraform execution logs
   - This workspace is automatically launched when the VSI is created
   - To access the workspace manually: Navigate to **IBM Cloud Menu** > **Platform Automation** > **Schematics** > **Terraform** > Select your workspace
   - The workspace logs provide detailed information about the deployment process and can help troubleshoot any issues

2. **Profile Not Available in Region**
   - The system automatically handles region-specific profile availability
   - Check the logs to see which profile was selected

3. **Timeout Errors**
   - The deployment includes extended timeouts (60 minutes) for instance creation
   - If issues persist, check IBM Cloud status

4. **Password Empty on First Boot**
   - This is normal behavior for catalog offerings
   - The password will be set after the first boot completes
   - Use the instance ID as the initial password


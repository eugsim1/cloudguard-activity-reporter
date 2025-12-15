
# CloudGuard Activity Reporter

This Go program fetches activity and detected problems from Oracle Cloud Guard and generates a CSV report along with a detailed summary.

---

## Features

- Fetches detected problems from Cloud Guard
- Filters by compartment, region, resource type, risk level, and problem ID
- Generates a CSV report
- Prints a detailed summary including recent problems
- Calculates days since detection

---

## Requirements

- Go 1.21+ (or latest)
- Oracle OCI Go SDK v65
- OCI CLI configured or environment variables for authentication

---

## Initialize Go Modules

After cloning the repository, run:

```bash
go mod tidy
````

This will download and install all required dependencies for the project.

---

## Usage

```bash
go run cloudguard_activity.go \
  -compartment-id=<compartment_ocid> \
  -days=7 \
  -region=<region> \
  -output=activity.csv
```

---

## Flags

| Flag              | Description                                                  |
| ----------------- | ------------------------------------------------------------ |
| `-compartment-id` | OCI Compartment OCID (required)                              |
| `-days`           | Number of days back to search (default: 7)                   |
| `-region`         | OCI region filter (optional)                                 |
| `-resource-type`  | Filter by resource type (optional)                           |
| `-problem-id`     | Filter by specific problem ID (optional)                     |
| `-risk-level`     | Filter by risk level: CRITICAL, HIGH, MEDIUM, LOW (optional) |
| `-limit`          | Maximum number of results (default: 1000)                    |
| `-summary`        | Print summary only, skip CSV export                          |
| `-output`         | Output CSV file path (default: `cloudguard_activity.csv`)    |

---

## Example

```bash
go run cloudguard_activity.go \
  -compartment-id ocid1.compartment.oc1..aaaaaaaaxxxxx \
  -days 5 \
  -region us-ashburn-1 \
  -risk-level HIGH \
  -output reports/high_risk_activity.csv
```

This command fetches high-risk Cloud Guard problems from the last 5 days in the `us-ashburn-1` region and saves the results to a CSV file.

---

## Summary Output (Example)

```
=== CLOUD GUARD ACTIVITY SUMMARY ===
Total problems detected: 7

By Risk Level:
  CRITICAL  : 2 (28.6%)
  HIGH      : 3 (42.9%)
  MEDIUM    : 2 (28.6%)

By Resource Type:
  Instance                  : 4
  Database                  : 2
  NetworkSecurityGroup      : 1

By Detector:
  ConfigurationDetector     : 3
  ActivityDetector          : 4

By Region:
  us-ashburn-1              : 5
  us-phoenix-1              : 2

Most Recent Problems:
  1. [12/15 10:24] Instance - Open SSH port issue (HIGH) - 2 days ago
  2. [12/14 08:12] Database - Weak password detected (CRITICAL) - 3 days ago
  3. [12/13 16:50] NetworkSecurityGroup - Excessive rules (MEDIUM) - 4 days ago
  4. [12/13 09:33] Instance - Misconfigured logging (HIGH) - 5 days ago
  5. [12/12 14:18] Database - Public access allowed (CRITICAL) - 6 days ago
```

---

## Sample CSV Output (Fake Data)

```
Problem_ID,First_Detected,Last_Detected,Days_Since_Detection,Resource_ID,Resource_Name,Resource_Type,Region,Compartment_ID,Detector,Risk_Level,Description,Recommendation,Detector_Rule_ID,Target_ID,Labels,Lifecycle_State
p1,2025-12-10T08:00:00Z,2025-12-15T10:24:00Z,5,r1,Instance-01,Instance,us-ashburn-1,ocid1.compartment.oc1..xxxx,ConfigurationDetector,HIGH,Open SSH port,Restrict SSH access,dr1,t1,SSH|Network,ACTIVE
p2,2025-12-12T09:00:00Z,2025-12-14T08:12:00Z,2,r2,DB-01,Database,us-ashburn-1,ocid1.compartment.oc1..xxxx,ActivityDetector,CRITICAL,Weak password detected,Enforce strong passwords,dr2,t2,Security|Compliance,ACTIVE
p3,2025-12-11T14:30:00Z,2025-12-13T16:50:00Z,2,r3,NSG-01,NetworkSecurityGroup,us-phoenix-1,ocid1.compartment.oc1..xxxx,ConfigurationDetector,MEDIUM,Excessive rules,Reduce number of rules,dr3,t3,Network|Firewall,ACTIVE
```

---



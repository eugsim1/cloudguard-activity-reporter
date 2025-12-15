# Cloud Guard Activity Reporter

A Go program to fetch and summarize **OCI Cloud Guard detected problems**.  
The tool generates a **CSV report** and prints a **detailed summary** of detected security issues across compartments, regions, and resources.

---

## Features

- Fetch Cloud Guard problems for a specific compartment and time range
- Filter by region, resource type, problem ID, and risk level
- Summarize results by risk level, resource type, detector, and region
- Include "days since detection" for each problem
- Export detailed results to CSV
- Optionally display summary only

---

## Prerequisites

- Go 1.21+ (or latest supported version)
- Oracle Cloud Infrastructure (OCI) CLI configured OR environment variables set
- OCI Go SDK v65:

```bash
go get github.com/oracle/oci-go-sdk/v65/cloudguard
go get github.com/oracle/oci-go-sdk/v65/common

/*
 go run cloudguard_activity.go -compartment-id=ocid1.tenancy.XXXXX -days=5

*/


package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/cloudguard"
)

type ActivityFilter struct {
	CompartmentID string
	Region        string
	StartTime     time.Time
	EndTime       time.Time
	ResourceType  string
	ProblemID     string
	RiskLevel     string
	Limit         int
}

type DetectedProblem struct {
	ID               string
	ResourceID       string
	ResourceName     string
	ResourceType     string
	Region           string
	CompartmentID    string
	CompartmentName  string
	Detector         string
	RiskLevel        string
	FirstDetected    time.Time
	LastDetected     time.Time
	DaysSinceDetection int // New field
	Description      string
	Recommendation   string
	DetectorRuleID   string
	DetectorRuleName string
	Labels           []string
	TargetID         string
	TargetName       string
	LifecycleState   string
}

type ActivitySummary struct {
	TotalProblems    int
	ByRiskLevel      map[string]int
	ByResourceType   map[string]int
	ByDetector       map[string]int
	ByRegion         map[string]int
	RecentProblems   []DetectedProblem
}

func main() {
	// Define command line flags
	compartmentID := flag.String("compartment-id", "", "Compartment ID (required)")
	outputFile := flag.String("output", "cloudguard_activity.csv", "Output CSV file")
	daysBack := flag.Int("days", 7, "Number of days back to search")
	region := flag.String("region", "", "Specific region to filter")
	resourceType := flag.String("resource-type", "", "Resource type filter")
	problemID := flag.String("problem-id", "", "Specific problem ID")
	riskLevel := flag.String("risk-level", "", "Risk level filter (CRITICAL, HIGH, MEDIUM, LOW)")
	limit := flag.Int("limit", 1000, "Maximum number of results")
	summaryOnly := flag.Bool("summary", false, "Print summary only (no CSV export)")
	flag.Parse()

	// Validate required flags
	if *compartmentID == "" {
		*compartmentID = os.Getenv("OCI_COMPARTMENT_ID")
		if *compartmentID == "" {
			fmt.Println("Error: compartment-id is required")
			fmt.Println("Usage: go run cloudguard_activity.go -compartment-id=ocid1.compartment.oc1..xxx [-output=activity.csv] [-days=7] [-region=us-ashburn-1] [-resource-type=Instance] [-problem-id=xxx] [-risk-level=HIGH] [-limit=1000] [-summary]")
			os.Exit(1)
		}
	}

	// Create Cloud Guard client
	client, err := cloudguard.NewCloudGuardClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		fmt.Printf("Error creating Cloud Guard client: %v\n", err)
		os.Exit(1)
	}

	// Set up time range
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -*daysBack)

	filter := ActivityFilter{
		CompartmentID: *compartmentID,
		Region:        *region,
		StartTime:     startTime,
		EndTime:       endTime,
		ResourceType:  *resourceType,
		ProblemID:     *problemID,
		RiskLevel:     *riskLevel,
		Limit:         *limit,
	}

	fmt.Printf("Searching Cloud Guard activity from %s to %s\n", 
		startTime.Format("2006-01-02"), 
		endTime.Format("2006-01-02"))
	fmt.Printf("Compartment: %s\n", *compartmentID)
	
	if *region != "" {
		fmt.Printf("Region: %s\n", *region)
	}
	if *resourceType != "" {
		fmt.Printf("Resource Type: %s\n", *resourceType)
	}
	if *riskLevel != "" {
		fmt.Printf("Risk Level: %s\n", *riskLevel)
	}

	// Get detected problems
	problems, err := getDetectedProblems(client, filter)
	if err != nil {
		fmt.Printf("Error getting detected problems: %v\n", err)
		os.Exit(1)
	}

	// Generate summary
	summary := generateSummary(problems)
	printDetailedSummary(summary)

	if *summaryOnly {
		fmt.Println("\nSummary only mode - skipping CSV export")
		return
	}

	// Export to CSV
	err = exportToCSV(problems, *outputFile)
	if err != nil {
		fmt.Printf("Error exporting to CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nActivity report saved to: %s\n", *outputFile)
}

func getDetectedProblems(client cloudguard.CloudGuardClient, filter ActivityFilter) ([]DetectedProblem, error) {
	var allProblems []DetectedProblem
	var page *string

	for {
		request := cloudguard.ListProblemsRequest{
			CompartmentId: common.String(filter.CompartmentID),
			Page:          page,
			Limit:         common.Int(filter.Limit),
		}

		response, err := client.ListProblems(context.Background(), request)
		if err != nil {
			return nil, fmt.Errorf("failed to list problems: %v", err)
		}

		// Process each problem using correct field names
		for _, problem := range response.Items {
			detectedProblem := DetectedProblem{
				ID:              safeString(problem.Id),
				ResourceID:      safeString(problem.ResourceId),
				ResourceName:    safeString(problem.ResourceName),
				ResourceType:    safeString(problem.ResourceType),
				Region:          safeString(problem.Region),
				CompartmentID:   safeString(problem.CompartmentId),
				RiskLevel:       string(problem.RiskLevel),
				DetectorRuleID:  safeString(problem.DetectorRuleId),
				Labels:          problem.Labels,
				TargetID:        safeString(problem.TargetId),
				LifecycleState:  string(problem.LifecycleState),
			}

			// Get detector from detector rule ID or use a default
			if problem.DetectorRuleId != nil {
				detectedProblem.Detector = extractDetectorFromRuleID(*problem.DetectorRuleId)
			}

			if problem.TimeFirstDetected != nil {
				detectedProblem.FirstDetected = problem.TimeFirstDetected.Time
			}
			if problem.TimeLastDetected != nil {
				detectedProblem.LastDetected = problem.TimeLastDetected.Time
			}

			// Calculate days since detection
			if !detectedProblem.FirstDetected.IsZero() {
				detectedProblem.DaysSinceDetection = int(time.Since(detectedProblem.FirstDetected).Hours() / 24)
			}

			// Apply manual filtering if filter parameters were provided
			if shouldIncludeProblem(detectedProblem, filter) {
				// Get additional details for the problem
				enrichedProblem, err := enrichProblemDetails(client, detectedProblem)
				if err != nil {
					fmt.Printf("Warning: Failed to enrich problem %s: %v\n", detectedProblem.ID, err)
					allProblems = append(allProblems, detectedProblem)
				} else {
					allProblems = append(allProblems, enrichedProblem)
				}
			}
		}

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage

		// Break if we've reached the limit
		if len(allProblems) >= filter.Limit {
			break
		}
	}

	return allProblems, nil
}

func extractDetectorFromRuleID(ruleID string) string {
	// Extract detector type from rule ID pattern
	// Example: ocid1.cloudguarddetectorrule.oc1..xxxxxx:ConfigurationDetector:xxxx
	parts := strings.Split(ruleID, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "UNKNOWN"
}

func shouldIncludeProblem(problem DetectedProblem, filter ActivityFilter) bool {
	// Manual filtering since Filter field might not be available in request
	if filter.Region != "" && problem.Region != filter.Region {
		return false
	}
	if filter.ResourceType != "" && problem.ResourceType != filter.ResourceType {
		return false
	}
	if filter.ProblemID != "" && problem.ID != filter.ProblemID {
		return false
	}
	if filter.RiskLevel != "" && problem.RiskLevel != filter.RiskLevel {
		return false
	}
	if !filter.StartTime.IsZero() && problem.LastDetected.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && problem.LastDetected.After(filter.EndTime) {
		return false
	}
	return true
}

func enrichProblemDetails(client cloudguard.CloudGuardClient, problem DetectedProblem) (DetectedProblem, error) {
	// Get problem details for additional information
	detailRequest := cloudguard.GetProblemRequest{
		ProblemId: common.String(problem.ID),
	}

	detailResponse, err := client.GetProblem(context.Background(), detailRequest)
	if err != nil {
		return problem, err
	}

	// Add description and recommendation from detailed response
	problem.Description = safeString(detailResponse.Description)
	problem.Recommendation = safeString(detailResponse.Recommendation)

	return problem, nil
}

func generateSummary(problems []DetectedProblem) ActivitySummary {
	summary := ActivitySummary{
		TotalProblems: len(problems),
		ByRiskLevel:   make(map[string]int),
		ByResourceType: make(map[string]int),
		ByDetector:     make(map[string]int),
		ByRegion:       make(map[string]int),
	}

	// Get recent problems (last 5)
	recentCount := 5
	if len(problems) < recentCount {
		recentCount = len(problems)
	}
	summary.RecentProblems = problems[:recentCount]

	for _, problem := range problems {
		summary.ByRiskLevel[problem.RiskLevel]++
		summary.ByResourceType[problem.ResourceType]++
		summary.ByDetector[problem.Detector]++
		summary.ByRegion[problem.Region]++
	}

	return summary
}

func printDetailedSummary(summary ActivitySummary) {
	fmt.Printf("\n=== CLOUD GUARD ACTIVITY SUMMARY ===\n")
	fmt.Printf("Total problems detected: %d\n", summary.TotalProblems)

	if summary.TotalProblems == 0 {
		return
	}

	fmt.Printf("\nBy Risk Level:\n")
	for level, count := range summary.ByRiskLevel {
		percentage := float64(count) / float64(summary.TotalProblems) * 100
		fmt.Printf("  %-10s: %3d (%5.1f%%)\n", level, count, percentage)
	}

	fmt.Printf("\nBy Resource Type:\n")
	for resourceType, count := range summary.ByResourceType {
		fmt.Printf("  %-25s: %3d\n", resourceType, count)
	}

	fmt.Printf("\nBy Detector:\n")
	for detector, count := range summary.ByDetector {
		fmt.Printf("  %-20s: %3d\n", detector, count)
	}

	fmt.Printf("\nBy Region:\n")
	for region, count := range summary.ByRegion {
		fmt.Printf("  %-20s: %3d\n", region, count)
	}

	fmt.Printf("\nMost Recent Problems:\n")
	for i, problem := range summary.RecentProblems {
		desc := problem.Description
		if desc == "N/A" {
			desc = problem.ResourceType + " issue"
		}
		fmt.Printf("  %d. [%s] %s - %s (%s) - %d days ago\n",
			i+1,
			problem.LastDetected.Format("01/02 15:04"),
			problem.ResourceType,
			truncateString(desc, 50),
			problem.RiskLevel,
			problem.DaysSinceDetection)
	}
}

func exportToCSV(problems []DetectedProblem, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := []string{
		"Problem_ID",
		"First_Detected",
		"Last_Detected",
		"Days_Since_Detection", // New column
		"Resource_ID",
		"Resource_Name",
		"Resource_Type",
		"Region",
		"Compartment_ID",
		"Detector",
		"Risk_Level",
		"Description",
		"Recommendation",
		"Detector_Rule_ID",
		"Target_ID",
		"Labels",
		"Lifecycle_State",
	}

	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data
	for _, problem := range problems {
		labels := strings.Join(problem.Labels, "|")
		if labels == "" {
			labels = "N/A"
		}

		row := []string{
			problem.ID,
			problem.FirstDetected.Format(time.RFC3339),
			problem.LastDetected.Format(time.RFC3339),
			strconv.Itoa(problem.DaysSinceDetection), // New field
			problem.ResourceID,
			problem.ResourceName,
			problem.ResourceType,
			problem.Region,
			problem.CompartmentID,
			problem.Detector,
			problem.RiskLevel,
			problem.Description,
			problem.Recommendation,
			problem.DetectorRuleID,
			problem.TargetID,
			labels,
			problem.LifecycleState,
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions
func safeString(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

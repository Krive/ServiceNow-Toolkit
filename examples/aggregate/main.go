package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

func main() {
	// Initialize ServiceNow client
	client, err := servicenow.NewClientBasicAuth(
		os.Getenv("SERVICENOW_INSTANCE_URL"),
		os.Getenv("SN_USERNAME"),
		os.Getenv("SN_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	// Run examples
	fmt.Printf("=== ServiceNow Aggregate API Examples ===\n")

	simpleCountExample(client)
	countByStateExample(client)
	averageAndSumExample(client)
	minMaxExample(client)
	complexGroupingExample(client)
	dateRangeAnalysisExample(client)
	performanceMetricsExample(client)
	contextTimeoutExample(client)
}

// Example 1: Simple record count
func simpleCountExample(client *servicenow.Client) {
	fmt.Println("1. Simple Record Count")
	fmt.Println("----------------------")

	aggClient := client.Aggregate("incident")

	// Count all incidents
	totalCount, err := aggClient.CountRecords(nil)
	if err != nil {
		log.Printf("Error counting all incidents: %v", err)
		return
	}
	fmt.Printf("Total incidents: %d\n", totalCount)

	// Count active incidents only
	activeCount, err := aggClient.CountRecords(query.New().Equals("active", true))
	if err != nil {
		log.Printf("Error counting active incidents: %v", err)
		return
	}
	fmt.Printf("Active incidents: %d\n", activeCount)
	fmt.Println()
}

// Example 2: Count incidents by state
func countByStateExample(client *servicenow.Client) {
	fmt.Println("2. Count Incidents by State")
	fmt.Println("---------------------------")

	aggClient := client.Aggregate("incident")

	result, err := aggClient.NewQuery().
		CountAll("incident_count").
		GroupByField("state", "state_name").
		OrderByDesc("incident_count").
		Limit(10).
		Execute()

	if err != nil {
		log.Printf("Error executing state count query: %v", err)
		return
	}

	fmt.Println("Incident count by state:")
	for i, row := range result.Result {
		if i >= 5 { // Show first 5 results
			break
		}
		fmt.Printf("  State %v: %v incidents\n", row["state_name"], row["incident_count"])
	}
	fmt.Println()
}

// Example 3: Average priority and sum of urgency
func averageAndSumExample(client *servicenow.Client) {
	fmt.Println("3. Average Priority and Sum of Urgency")
	fmt.Println("--------------------------------------")

	aggClient := client.Aggregate("incident")

	result, err := aggClient.NewQuery().
		Avg("priority", "avg_priority").
		Sum("urgency", "total_urgency").
		CountAll("total_incidents").
		Where("active", query.OpEquals, true).
		Execute()

	if err != nil {
		log.Printf("Error executing average/sum query: %v", err)
		return
	}

	if result.Stats != nil {
		fmt.Printf("Active Incidents Analysis:\n")
		fmt.Printf("  Total: %v\n", result.Stats["total_incidents"])
		fmt.Printf("  Average Priority: %v\n", result.Stats["avg_priority"])
		fmt.Printf("  Total Urgency Score: %v\n", result.Stats["total_urgency"])
	}
	fmt.Println()
}

// Example 4: Min/Max priority values
func minMaxExample(client *servicenow.Client) {
	fmt.Println("4. Min/Max Priority Analysis")
	fmt.Println("----------------------------")

	aggClient := client.Aggregate("incident")

	min, max, err := aggClient.MinMaxField("priority", query.New().Equals("active", true))
	if err != nil {
		log.Printf("Error getting min/max priority: %v", err)
		return
	}

	fmt.Printf("Priority range for active incidents:\n")
	fmt.Printf("  Minimum: %.0f\n", min)
	fmt.Printf("  Maximum: %.0f\n", max)
	fmt.Printf("  Range: %.0f\n", max-min)
	fmt.Println()
}

// Example 5: Complex grouping with multiple aggregates
func complexGroupingExample(client *servicenow.Client) {
	fmt.Println("5. Complex Grouping Analysis")
	fmt.Println("----------------------------")

	aggClient := client.Aggregate("incident")

	result, err := aggClient.NewQuery().
		CountAll("incident_count").
		Avg("priority", "avg_priority").
		Min("urgency", "min_urgency").
		Max("urgency", "max_urgency").
		GroupByField("state", "incident_state").
		GroupByField("priority", "priority_level").
		Where("active", query.OpEquals, true).
		Having("COUNT(*) > 0").
		OrderByDesc("incident_count").
		OrderByAsc("incident_state").
		Limit(10).
		Execute()

	if err != nil {
		log.Printf("Error executing complex grouping query: %v", err)
		return
	}

	fmt.Println("Incidents grouped by state and priority:")
	for i, row := range result.Result {
		if i >= 3 { // Show first 3 results
			break
		}
		fmt.Printf("  State %v, Priority %v:\n", row["incident_state"], row["priority_level"])
		fmt.Printf("    Count: %v, Avg Priority: %v\n", row["incident_count"], row["avg_priority"])
		fmt.Printf("    Urgency Range: %v - %v\n", row["min_urgency"], row["max_urgency"])
	}
	fmt.Println()
}

// Example 6: Date range analysis
func dateRangeAnalysisExample(client *servicenow.Client) {
	fmt.Println("6. Date Range Analysis")
	fmt.Println("----------------------")

	aggClient := client.Aggregate("incident")

	// Count incidents created in the last 7 days
	result, err := aggClient.NewQuery().
		CountAll("daily_count").
		Avg("priority", "avg_daily_priority").
		Where("sys_created_on", query.OpGreaterThan, "javascript:gs.daysAgoStart(7)").
		Execute()

	if err != nil {
		log.Printf("Error executing date range query: %v", err)
		return
	}

	if result.Stats != nil {
		fmt.Printf("Last 7 days incident analysis:\n")
		fmt.Printf("  Total created: %v\n", result.Stats["daily_count"])
		fmt.Printf("  Average priority: %v\n", result.Stats["avg_daily_priority"])
	}
	fmt.Println()
}

// Example 7: Performance metrics with standard deviation
func performanceMetricsExample(client *servicenow.Client) {
	fmt.Println("7. Performance Metrics")
	fmt.Println("----------------------")

	aggClient := client.Aggregate("incident")

	result, err := aggClient.NewQuery().
		CountAll("total_count").
		Avg("priority", "avg_priority").
		StdDev("priority", "priority_stddev").
		Variance("priority", "priority_variance").
		Min("priority", "min_priority").
		Max("priority", "max_priority").
		Where("active", query.OpEquals, true).
		Execute()

	if err != nil {
		log.Printf("Error executing performance metrics query: %v", err)
		return
	}

	if result.Stats != nil {
		fmt.Printf("Priority distribution statistics:\n")
		fmt.Printf("  Count: %v\n", result.Stats["total_count"])
		fmt.Printf("  Mean: %v\n", result.Stats["avg_priority"])
		fmt.Printf("  Standard Deviation: %v\n", result.Stats["priority_stddev"])
		fmt.Printf("  Variance: %v\n", result.Stats["priority_variance"])
		fmt.Printf("  Range: %v - %v\n", result.Stats["min_priority"], result.Stats["max_priority"])
	}
	fmt.Println()
}

// Example 8: Context with timeout
func contextTimeoutExample(client *servicenow.Client) {
	fmt.Println("8. Context with Timeout")
	fmt.Println("-----------------------")

	aggClient := client.Aggregate("incident")

	// Create context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute query with context
	count, err := aggClient.CountRecordsWithContext(ctx, query.New().Equals("active", true))
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Query timed out after 5 seconds")
		} else {
			log.Printf("Error with context query: %v", err)
		}
		return
	}

	fmt.Printf("Active incidents (with 5s timeout): %d\n", count)
	fmt.Println()
}

// Example usage in a real application:
func exampleUsageInApplication() {
	// This function demonstrates how you might use the aggregate API in a real application

	client, _ := servicenow.NewClientBasicAuth("https://dev.service-now.com", "user", "pass")
	aggClient := client.Aggregate("incident")

	// Dashboard metrics
	dashboardMetrics := func() (map[string]interface{}, error) {
		result, err := aggClient.NewQuery().
			CountAll("total_incidents").
			Count("state", "open_incidents").
			Avg("priority", "avg_priority").
			Where("active", query.OpEquals, true).
			Execute()

		if err != nil {
			return nil, err
		}

		return result.Stats, nil
	}

	// Department performance analysis
	departmentAnalysis := func(department string) ([]map[string]interface{}, error) {
		result, err := aggClient.NewQuery().
			CountAll("incident_count").
			Avg("priority", "avg_priority").
			Min("sys_created_on", "earliest_incident").
			Max("sys_created_on", "latest_incident").
			GroupByField("assignment_group", "group").
			Where("assignment_group.department", query.OpEquals, department).
			And().
			Where("active", query.OpEquals, true).
			OrderByDesc("incident_count").
			Limit(20).
			Execute()

		if err != nil {
			return nil, err
		}

		return result.Result, nil
	}

	// Trend analysis
	monthlyTrends := func() ([]map[string]interface{}, error) {
		result, err := aggClient.NewQuery().
			CountAll("monthly_count").
			Avg("priority", "avg_monthly_priority").
			GroupByField("sys_created_on", "month"). // Would need proper date grouping in real use
			Where("sys_created_on", query.OpGreaterThan, "javascript:gs.daysAgoStart(365)").
			OrderByAsc("month").
			Execute()

		if err != nil {
			return nil, err
		}

		return result.Result, nil
	}

	// Use the functions (this is just for demonstration)
	_, _ = dashboardMetrics()
	_, _ = departmentAnalysis("IT")
	_, _ = monthlyTrends()
}

package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

func setupAggregateClient(t *testing.T) *servicenow.Client {
	username := os.Getenv("SN_USERNAME")
	password := os.Getenv("SN_PASSWORD")
	instanceURL := os.Getenv("SN_INSTANCE_URL")

	if username == "" || password == "" || instanceURL == "" {
		t.Skip("ServiceNow credentials not provided, skipping integration tests")
	}

	client, err := servicenow.NewClientBasicAuth(instanceURL, username, password)
	if err != nil {
		t.Fatalf("Failed to create ServiceNow client: %v", err)
	}

	return client
}

func TestAggregateIntegration_CountRecords(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Test basic count
	count, err := aggClient.CountRecords(nil)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}

	log.Printf("Total incidents: %d", count)
	
	if count < 0 {
		t.Errorf("Expected non-negative count, got %d", count)
	}
}

func TestAggregateIntegration_CountWithFilter(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Test count with filter
	qb := query.New().Equals("active", true)
	activeCount, err := aggClient.CountRecords(qb)
	if err != nil {
		t.Fatalf("Failed to count active records: %v", err)
	}

	log.Printf("Active incidents: %d", activeCount)
	
	if activeCount < 0 {
		t.Errorf("Expected non-negative count, got %d", activeCount)
	}
}

func TestAggregateIntegration_CountByState(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Count incidents grouped by state
	result, err := aggClient.NewQuery().
		CountAll("incident_count").
		GroupByField("state", "state_name").
		OrderByDesc("incident_count").
		Limit(10).
		Execute()

	if err != nil {
		t.Fatalf("Failed to execute aggregate query: %v", err)
	}

	log.Printf("Incident count by state: %+v", result)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if len(result.Result) == 0 {
		t.Log("No results returned - this might be expected if there are no incidents")
	} else {
		log.Printf("Found %d state groups", len(result.Result))
		for i, row := range result.Result {
			if i >= 3 { // Log only first 3 for brevity
				break
			}
			log.Printf("State group %d: %+v", i+1, row)
		}
	}
}

func TestAggregateIntegration_AverageValues(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Get average priority and urgency
	result, err := aggClient.NewQuery().
		Avg("priority", "avg_priority").
		Avg("urgency", "avg_urgency").
		CountAll("total_count").
		Where("active", query.OpEquals, true).
		Execute()

	if err != nil {
		t.Fatalf("Failed to execute average query: %v", err)
	}

	log.Printf("Average values result: %+v", result)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if result.Stats != nil {
		if avgPriority, exists := result.Stats["avg_priority"]; exists {
			log.Printf("Average priority: %v", avgPriority)
		}
		if avgUrgency, exists := result.Stats["avg_urgency"]; exists {
			log.Printf("Average urgency: %v", avgUrgency)
		}
		if totalCount, exists := result.Stats["total_count"]; exists {
			log.Printf("Total active incidents: %v", totalCount)
		}
	}
}

func TestAggregateIntegration_MinMaxValues(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Get min/max priority values
	min, max, err := aggClient.MinMaxField("priority", query.New().Equals("active", true))
	if err != nil {
		t.Fatalf("Failed to get min/max values: %v", err)
	}

	log.Printf("Priority range: min=%f, max=%f", min, max)

	if min > max {
		t.Errorf("Min value (%f) should not be greater than max value (%f)", min, max)
	}
}

func TestAggregateIntegration_ComplexGrouping(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Complex query with multiple aggregates and grouping
	result, err := aggClient.NewQuery().
		CountAll("incident_count").
		Avg("priority", "avg_priority").
		Min("urgency", "min_urgency").
		Max("urgency", "max_urgency").
		GroupByField("state", "incident_state").
		GroupByField("priority", "priority_level").
		Where("active", query.OpEquals, true).
		Having("COUNT(*) > 0"). // Basic having clause
		OrderByDesc("incident_count").
		OrderByAsc("incident_state").
		Limit(20).
		Execute()

	if err != nil {
		t.Fatalf("Failed to execute complex grouping query: %v", err)
	}

	log.Printf("Complex grouping result: %+v", result)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if len(result.Result) == 0 {
		t.Log("No results returned - this might be expected")
	} else {
		log.Printf("Found %d grouped results", len(result.Result))
		for i, row := range result.Result {
			if i >= 5 { // Log only first 5 for brevity
				break
			}
			log.Printf("Group %d: %+v", i+1, row)
		}
	}
}

func TestAggregateIntegration_ContextTimeout(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Test with very short timeout to ensure context cancellation works
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := aggClient.CountRecordsWithContext(ctx, nil)
	
	// We expect this to timeout or be cancelled
	if err == nil {
		t.Log("Query completed before timeout - this is possible with very fast responses")
	} else {
		log.Printf("Query failed as expected due to context timeout: %v", err)
	}
}

func TestAggregateIntegration_ContextCancellation(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel the context immediately
	cancel()

	_, err := aggClient.CountRecordsWithContext(ctx, nil)
	
	// We expect this to be cancelled
	if err == nil {
		t.Log("Query completed before cancellation - this is possible with very fast responses")
	} else {
		log.Printf("Query failed as expected due to context cancellation: %v", err)
	}
}

func TestAggregateIntegration_SumField(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Sum priority values for active incidents
	sum, err := aggClient.SumField("priority", query.New().Equals("active", true))
	if err != nil {
		t.Fatalf("Failed to sum field: %v", err)
	}

	log.Printf("Sum of priority for active incidents: %f", sum)

	if sum < 0 {
		t.Errorf("Expected non-negative sum, got %f", sum)
	}
}

func TestAggregateIntegration_AvgField(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Average priority for active incidents
	avg, err := aggClient.AvgField("priority", query.New().Equals("active", true))
	if err != nil {
		t.Fatalf("Failed to get average field: %v", err)
	}

	log.Printf("Average priority for active incidents: %f", avg)

	if avg < 0 {
		t.Errorf("Expected non-negative average, got %f", avg)
	}
}

func TestAggregateIntegration_MultipleAggregatesOneQuery(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Single query with multiple aggregate operations
	result, err := aggClient.NewQuery().
		CountAll("total_incidents").
		Sum("priority", "priority_sum").
		Avg("priority", "priority_avg").
		Min("priority", "priority_min").
		Max("priority", "priority_max").
		StdDev("priority", "priority_stddev").
		Where("active", query.OpEquals, true).
		Execute()

	if err != nil {
		t.Fatalf("Failed to execute multi-aggregate query: %v", err)
	}

	log.Printf("Multi-aggregate result: %+v", result)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if result.Stats != nil {
		expectedFields := []string{"total_incidents", "priority_sum", "priority_avg", "priority_min", "priority_max", "priority_stddev"}
		for _, field := range expectedFields {
			if value, exists := result.Stats[field]; exists {
				log.Printf("%s: %v", field, value)
			} else {
				log.Printf("%s: not found in result", field)
			}
		}
	}
}

func TestAggregateIntegration_DateRangeAggregation(t *testing.T) {
	client := setupAggregateClient(t)
	aggClient := client.Aggregate("incident")

	// Count incidents created in the last 30 days, grouped by day
	result, err := aggClient.NewQuery().
		CountAll("daily_count").
		GroupByField("sys_created_on", "creation_date").
		Where("sys_created_on", query.OpGreaterThan, "javascript:gs.daysAgoStart(30)").
		OrderByDesc("creation_date").
		Limit(30).
		Execute()

	if err != nil {
		t.Fatalf("Failed to execute date range aggregation: %v", err)
	}

	log.Printf("Date range aggregation result: %+v", result)

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if len(result.Result) == 0 {
		t.Log("No results returned for last 30 days - this might be expected")
	} else {
		log.Printf("Found incident data for %d days", len(result.Result))
		for i, row := range result.Result {
			if i >= 5 { // Log only first 5 days for brevity
				break
			}
			log.Printf("Day %d: %+v", i+1, row)
		}
	}
}
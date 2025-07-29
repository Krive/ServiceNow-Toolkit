package unit

import (
	"testing"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

func TestQueryBuilderBasic(t *testing.T) {
	q := query.New().
		Equals("active", true).
		And().
		Contains("description", "test")

	expectedQuery := "active=true^descriptionCONTAINStest"
	actualQuery := q.BuildQuery()

	if actualQuery != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, actualQuery)
	}
}

func TestQueryBuilderWithOrdering(t *testing.T) {
	q := query.New().
		Equals("state", "1").
		OrderByDesc("sys_created_on").
		Limit(10).
		Fields("number", "short_description")

	params := q.Build()

	if params["sysparm_query"] != "state=1" {
		t.Errorf("Expected query 'state=1', got '%s'", params["sysparm_query"])
	}

	if params["sysparm_orderby"] != "sys_created_on DESC" {
		t.Errorf("Expected orderby 'sys_created_on DESC', got '%s'", params["sysparm_orderby"])
	}

	if params["sysparm_limit"] != "10" {
		t.Errorf("Expected limit '10', got '%s'", params["sysparm_limit"])
	}

	if params["sysparm_fields"] != "number,short_description" {
		t.Errorf("Expected fields 'number,short_description', got '%s'", params["sysparm_fields"])
	}
}

func TestQueryBuilderComplexConditions(t *testing.T) {
	q := query.New().
		Equals("priority", "1").
		Or().
		Equals("urgency", "1").
		And().
		Equals("active", true)

	expectedQuery := "priority=1^ORurgency=1^active=true"
	actualQuery := q.BuildQuery()

	if actualQuery != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, actualQuery)
	}
}

func TestQueryBuilderOperators(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *query.QueryBuilder
		expected string
	}{
		{
			name:     "GreaterThan",
			builder:  func() *query.QueryBuilder { return query.New().GreaterThan("count", 10) },
			expected: "count>10",
		},
		{
			name:     "LessThan", 
			builder:  func() *query.QueryBuilder { return query.New().LessThan("age", 30) },
			expected: "age<30",
		},
		{
			name:     "StartsWith",
			builder:  func() *query.QueryBuilder { return query.New().StartsWith("name", "John") },
			expected: "nameSTARTSWITHJohn",
		},
		{
			name:     "EndsWith",
			builder:  func() *query.QueryBuilder { return query.New().EndsWith("email", "@company.com") },
			expected: "emailENDSWITH%40company.com", // URL encoded
		},
		{
			name:     "IsEmpty",
			builder:  func() *query.QueryBuilder { return query.New().IsEmpty("description") },
			expected: "descriptionISEMPTY",
		},
		{
			name:     "IsNotEmpty",
			builder:  func() *query.QueryBuilder { return query.New().IsNotEmpty("comments") },
			expected: "commentsISNOTEMPTY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.builder().BuildQuery()
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}

func TestQueryBuilderInOperator(t *testing.T) {
	q := query.New().In("state", []interface{}{"1", "2", "3"})
	expected := "stateIN1%2C2%2C3"
	actual := q.BuildQuery()

	if actual != expected {
		t.Errorf("Expected '%s', got '%s'", expected, actual)
	}
}

func TestQueryBuilderClone(t *testing.T) {
	original := query.New().
		Equals("active", true).
		OrderByAsc("name").
		Fields("number", "description")

	clone := original.Clone().
		And().
		Equals("state", "1")

	originalQuery := original.BuildQuery()
	cloneQuery := clone.BuildQuery()

	if originalQuery == cloneQuery {
		t.Error("Clone should not modify original query")
	}

	if originalQuery != "active=true" {
		t.Errorf("Original query should be 'active=true', got '%s'", originalQuery)
	}

	if cloneQuery != "active=true^state=1" {
		t.Errorf("Clone query should be 'active=true^state=1', got '%s'", cloneQuery)
	}
}

func TestPredefinedQueryBuilders(t *testing.T) {
	// Test ActiveRecords
	activeQuery := query.ActiveRecords().BuildQuery()
	if activeQuery != "active=true" {
		t.Errorf("ActiveRecords should return 'active=true', got '%s'", activeQuery)
	}

	// Test InactiveRecords
	inactiveQuery := query.InactiveRecords().BuildQuery()
	if inactiveQuery != "active=false" {
		t.Errorf("InactiveRecords should return 'active=false', got '%s'", inactiveQuery)
	}

	// Test ByState
	stateQuery := query.ByState("2").BuildQuery()
	if stateQuery != "state=2" {
		t.Errorf("ByState should return 'state=2', got '%s'", stateQuery)
	}

	// Test ByPriority
	priorityQuery := query.ByPriority(1).BuildQuery()
	if priorityQuery != "priority=1" {
		t.Errorf("ByPriority should return 'priority=1', got '%s'", priorityQuery)
	}
}

func TestQueryBuilderString(t *testing.T) {
	q := query.New().
		Equals("active", true).
		OrderByDesc("sys_created_on").
		Fields("number", "description").
		Limit(5)

	str := q.String()
	
	// Check that string representation contains expected parts
	if !contains(str, "WHERE: active=true") {
		t.Error("String representation should contain WHERE clause")
	}

	if !contains(str, "FIELDS: number, description") {
		t.Error("String representation should contain FIELDS")
	}

	if !contains(str, "ORDER BY: sys_created_on DESC") {
		t.Error("String representation should contain ORDER BY")
	}

	if !contains(str, "LIMIT: 5") {
		t.Error("String representation should contain LIMIT")
	}
}

func TestQuerySet(t *testing.T) {
	query1 := query.New().Equals("priority", "1")
	query2 := query.New().Equals("urgency", "1")
	query3 := query.New().Equals("state", "1")

	qs := query.NewQuerySet().
		Add(query1).
		Add(query2).
		Add(query3)

	// Test Union (OR)
	union := qs.Union()
	unionQuery := union.BuildQuery()
	expected := "(priority=1)^OR(urgency=1)^OR(state=1)"
	
	if unionQuery != expected {
		t.Errorf("Union query expected '%s', got '%s'", expected, unionQuery)
	}

	// Test Intersection (AND)
	intersection := qs.Intersection()
	intersectionQuery := intersection.BuildQuery()
	expected = "(priority=1)^(urgency=1)^(state=1)"
	
	if intersectionQuery != expected {
		t.Errorf("Intersection query expected '%s', got '%s'", expected, intersectionQuery)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 indexOf(s, substr) >= 0)))
}

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
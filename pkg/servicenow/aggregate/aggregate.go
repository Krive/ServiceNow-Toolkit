package aggregate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/core"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

// AggregateClient handles ServiceNow Aggregate API operations
type AggregateClient struct {
	client    *core.Client
	tableName string
}

// NewAggregateClient creates a new aggregate client for a specific table
func NewAggregateClient(client *core.Client, tableName string) *AggregateClient {
	return &AggregateClient{
		client:    client,
		tableName: tableName,
	}
}

// AggregateType represents the type of aggregation
type AggregateType string

const (
	Count   AggregateType = "COUNT"
	Sum     AggregateType = "SUM"
	Avg     AggregateType = "AVG"
	Min     AggregateType = "MIN"
	Max     AggregateType = "MAX"
	StdDev  AggregateType = "STDDEV"
	Variance AggregateType = "VARIANCE"
)

// AggregateField represents a field to aggregate
type AggregateField struct {
	Field string        `json:"field"`
	Type  AggregateType `json:"type"`
	Alias string        `json:"alias,omitempty"`
}

// GroupBy represents a field to group by
type GroupBy struct {
	Field string `json:"field"`
	Alias string `json:"alias,omitempty"`
}

// AggregateQuery represents a complete aggregate query
type AggregateQuery struct {
	client       *AggregateClient
	aggregates   []AggregateField
	groupBy      []GroupBy
	having       []string
	queryBuilder *query.QueryBuilder
	orderBy      []string
	limit        int
	offset       int
}

// AggregateResult represents the result of an aggregate query
type AggregateResult struct {
	Stats  map[string]interface{} `json:"stats"`
	Result []map[string]interface{} `json:"result"`
}

// NewQuery creates a new aggregate query for the table
func (ac *AggregateClient) NewQuery() *AggregateQuery {
	return &AggregateQuery{
		client:       ac,
		aggregates:   make([]AggregateField, 0),
		groupBy:      make([]GroupBy, 0),
		having:       make([]string, 0),
		queryBuilder: query.New(),
		orderBy:      make([]string, 0),
		limit:        0,
		offset:       0,
	}
}

// Aggregate adds an aggregate field to the query
func (aq *AggregateQuery) Aggregate(field string, aggType AggregateType, alias string) *AggregateQuery {
	aq.aggregates = append(aq.aggregates, AggregateField{
		Field: field,
		Type:  aggType,
		Alias: alias,
	})
	return aq
}

// Count adds a COUNT aggregation
func (aq *AggregateQuery) Count(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Count, alias)
}

// CountAll adds a COUNT(*) aggregation
func (aq *AggregateQuery) CountAll(alias string) *AggregateQuery {
	return aq.Aggregate("", Count, alias)
}

// Sum adds a SUM aggregation
func (aq *AggregateQuery) Sum(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Sum, alias)
}

// Avg adds an AVG aggregation
func (aq *AggregateQuery) Avg(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Avg, alias)
}

// Min adds a MIN aggregation
func (aq *AggregateQuery) Min(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Min, alias)
}

// Max adds a MAX aggregation
func (aq *AggregateQuery) Max(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Max, alias)
}

// StdDev adds a STDDEV aggregation
func (aq *AggregateQuery) StdDev(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, StdDev, alias)
}

// Variance adds a VARIANCE aggregation
func (aq *AggregateQuery) Variance(field string, alias string) *AggregateQuery {
	return aq.Aggregate(field, Variance, alias)
}

// GroupByField adds a GROUP BY field
func (aq *AggregateQuery) GroupByField(field string, alias string) *AggregateQuery {
	aq.groupBy = append(aq.groupBy, GroupBy{
		Field: field,
		Alias: alias,
	})
	return aq
}

// Where adds a WHERE condition using the query builder
func (aq *AggregateQuery) Where(field string, operator query.Operator, value interface{}) *AggregateQuery {
	aq.queryBuilder.Where(field, operator, value)
	return aq
}

// And adds an AND condition
func (aq *AggregateQuery) And() *AggregateQuery {
	aq.queryBuilder.And()
	return aq
}

// Or adds an OR condition
func (aq *AggregateQuery) Or() *AggregateQuery {
	aq.queryBuilder.Or()
	return aq
}

// Equals adds an equality condition
func (aq *AggregateQuery) Equals(field string, value interface{}) *AggregateQuery {
	aq.queryBuilder.Equals(field, value)
	return aq
}

// Contains adds a contains condition
func (aq *AggregateQuery) Contains(field string, value interface{}) *AggregateQuery {
	aq.queryBuilder.Contains(field, value)
	return aq
}

// Having adds a HAVING condition for aggregate results
func (aq *AggregateQuery) Having(condition string) *AggregateQuery {
	aq.having = append(aq.having, condition)
	return aq
}

// OrderBy adds ordering to the aggregate results
func (aq *AggregateQuery) OrderBy(field string, direction query.OrderDirection) *AggregateQuery {
	orderStr := fmt.Sprintf("%s %s", field, string(direction))
	aq.orderBy = append(aq.orderBy, orderStr)
	return aq
}

// OrderByAsc adds ascending order
func (aq *AggregateQuery) OrderByAsc(field string) *AggregateQuery {
	return aq.OrderBy(field, query.OrderAsc)
}

// OrderByDesc adds descending order
func (aq *AggregateQuery) OrderByDesc(field string) *AggregateQuery {
	return aq.OrderBy(field, query.OrderDesc)
}

// Limit sets the maximum number of aggregate results
func (aq *AggregateQuery) Limit(limit int) *AggregateQuery {
	aq.limit = limit
	return aq
}

// Offset sets the number of aggregate results to skip
func (aq *AggregateQuery) Offset(offset int) *AggregateQuery {
	aq.offset = offset
	return aq
}

// Execute runs the aggregate query
func (aq *AggregateQuery) Execute() (*AggregateResult, error) {
	return aq.ExecuteWithContext(context.Background())
}

// ExecuteWithContext runs the aggregate query with context support
func (aq *AggregateQuery) ExecuteWithContext(ctx context.Context) (*AggregateResult, error) {
	params := aq.BuildParams()
	
	var result core.Response
	err := aq.client.client.RawRequestWithContext(ctx, "GET", fmt.Sprintf("/stats/%s", aq.client.tableName), nil, params, &result)
	if err != nil {
		return nil, fmt.Errorf("aggregate query failed: %w", err)
	}

	// Parse the response
	aggregateResult := &AggregateResult{}
	
	// Handle stats response format - ServiceNow returns nested structure
	if resultData, ok := result.Result.(map[string]interface{}); ok {
		// Check for nested stats structure: result.stats
		if stats, ok := resultData["stats"].(map[string]interface{}); ok {
			aggregateResult.Stats = stats
		} else {
			// Fallback: treat result directly as stats
			aggregateResult.Stats = resultData
		}
		
		// If there are group by fields, the result will be in a different format
		if len(aq.groupBy) > 0 {
			if resultArray, ok := resultData["result"].([]interface{}); ok {
				aggregateResult.Result = make([]map[string]interface{}, len(resultArray))
				for i, item := range resultArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						aggregateResult.Result[i] = itemMap
					}
				}
			}
		}
	} else if resultArray, ok := result.Result.([]interface{}); ok {
		// Handle array response format
		aggregateResult.Result = make([]map[string]interface{}, len(resultArray))
		for i, item := range resultArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				aggregateResult.Result[i] = itemMap
			}
		}
	}

	return aggregateResult, nil
}

// BuildParams constructs the query parameters for the aggregate request
func (aq *AggregateQuery) BuildParams() map[string]string {
	params := make(map[string]string)

	// Add WHERE conditions from query builder
	if queryStr := aq.queryBuilder.BuildQuery(); queryStr != "" {
		params["sysparm_query"] = queryStr
	}

	// Add aggregate fields using correct ServiceNow parameters
	if len(aq.aggregates) > 0 {
		var countFields []string
		var sumFields []string
		var avgFields []string
		var minFields []string
		var maxFields []string
		var stddevFields []string
		var varianceFields []string
		
		for _, agg := range aq.aggregates {
			fieldName := agg.Field
			
			switch agg.Type {
			case Count:
				if agg.Field == "" {
					// COUNT(*) case
					params["sysparm_count"] = "true"
				} else {
					countFields = append(countFields, fieldName)
				}
			case Sum:
				sumFields = append(sumFields, fieldName)
			case Avg:
				avgFields = append(avgFields, fieldName)
			case Min:
				minFields = append(minFields, fieldName)
			case Max:
				maxFields = append(maxFields, fieldName)
			case StdDev:
				stddevFields = append(stddevFields, fieldName)
			case Variance:
				varianceFields = append(varianceFields, fieldName)
			}
		}
		
		// Set the appropriate parameters
		if len(countFields) > 0 {
			params["sysparm_count_fields"] = strings.Join(countFields, ",")
		}
		if len(sumFields) > 0 {
			params["sysparm_sum_fields"] = strings.Join(sumFields, ",")
		}
		if len(avgFields) > 0 {
			params["sysparm_avg_fields"] = strings.Join(avgFields, ",")
		}
		if len(minFields) > 0 {
			params["sysparm_min_fields"] = strings.Join(minFields, ",")
		}
		if len(maxFields) > 0 {
			params["sysparm_max_fields"] = strings.Join(maxFields, ",")
		}
		if len(stddevFields) > 0 {
			params["sysparm_stddev_fields"] = strings.Join(stddevFields, ",")
		}
		if len(varianceFields) > 0 {
			params["sysparm_variance_fields"] = strings.Join(varianceFields, ",")
		}
	}

	// Add group by fields
	if len(aq.groupBy) > 0 {
		var groupParts []string
		for _, group := range aq.groupBy {
			groupStr := group.Field
			if group.Alias != "" {
				groupStr = fmt.Sprintf("%s AS %s", group.Field, group.Alias)
			}
			groupParts = append(groupParts, groupStr)
		}
		params["sysparm_group_by"] = strings.Join(groupParts, ",")
	}

	// Add having conditions
	if len(aq.having) > 0 {
		params["sysparm_having"] = strings.Join(aq.having, "^")
	}

	// Add ordering
	if len(aq.orderBy) > 0 {
		params["sysparm_orderby"] = strings.Join(aq.orderBy, ",")
	}

	// Add limit
	if aq.limit > 0 {
		params["sysparm_limit"] = strconv.Itoa(aq.limit)
	}

	// Add offset
	if aq.offset > 0 {
		params["sysparm_offset"] = strconv.Itoa(aq.offset)
	}

	return params
}

// Convenience methods for common aggregate operations

// CountRecords returns the total number of records matching the query
func (ac *AggregateClient) CountRecords(qb *query.QueryBuilder) (int, error) {
	return ac.CountRecordsWithContext(context.Background(), qb)
}

// CountRecordsWithContext returns the total number of records with context support
func (ac *AggregateClient) CountRecordsWithContext(ctx context.Context, qb *query.QueryBuilder) (int, error) {
	aq := ac.NewQuery().CountAll("count")
	if qb != nil {
		// Copy conditions from query builder
		aq.queryBuilder = qb.Clone()
	}

	result, err := aq.ExecuteWithContext(ctx)
	if err != nil {
		return 0, err
	}

	// Extract count from result - ServiceNow returns it as "count"
	if result.Stats != nil {
		if count, ok := result.Stats["count"]; ok {
			return parseIntFromInterface(count), nil
		}
	}

	return 0, fmt.Errorf("count not found in aggregate result")
}

// SumField returns the sum of a numeric field
func (ac *AggregateClient) SumField(field string, qb *query.QueryBuilder) (float64, error) {
	return ac.SumFieldWithContext(context.Background(), field, qb)
}

// SumFieldWithContext returns the sum of a numeric field with context support
func (ac *AggregateClient) SumFieldWithContext(ctx context.Context, field string, qb *query.QueryBuilder) (float64, error) {
	aq := ac.NewQuery().Sum(field, field)
	if qb != nil {
		aq.queryBuilder = qb.Clone()
	}

	result, err := aq.ExecuteWithContext(ctx)
	if err != nil {
		return 0, err
	}

	// Extract sum from result - ServiceNow returns nested structure: stats.sum.fieldname
	if result.Stats != nil {
		if sumObj, ok := result.Stats["sum"].(map[string]interface{}); ok {
			if sum, ok := sumObj[field]; ok {
				return parseFloatFromInterface(sum), nil
			}
		}
	}

	return 0, fmt.Errorf("sum for field '%s' not found in aggregate result", field)
}

// AvgField returns the average of a numeric field
func (ac *AggregateClient) AvgField(field string, qb *query.QueryBuilder) (float64, error) {
	return ac.AvgFieldWithContext(context.Background(), field, qb)
}

// AvgFieldWithContext returns the average of a numeric field with context support
func (ac *AggregateClient) AvgFieldWithContext(ctx context.Context, field string, qb *query.QueryBuilder) (float64, error) {
	aq := ac.NewQuery().Avg(field, field)
	if qb != nil {
		aq.queryBuilder = qb.Clone()
	}

	result, err := aq.ExecuteWithContext(ctx)
	if err != nil {
		return 0, err
	}

	// Extract average from result - ServiceNow returns nested structure: stats.avg.fieldname
	if result.Stats != nil {
		if avgObj, ok := result.Stats["avg"].(map[string]interface{}); ok {
			if avg, ok := avgObj[field]; ok {
				return parseFloatFromInterface(avg), nil
			}
		}
	}

	return 0, fmt.Errorf("average for field '%s' not found in aggregate result", field)
}

// MinMaxField returns both minimum and maximum values of a field
func (ac *AggregateClient) MinMaxField(field string, qb *query.QueryBuilder) (min, max float64, err error) {
	return ac.MinMaxFieldWithContext(context.Background(), field, qb)
}

// MinMaxFieldWithContext returns both minimum and maximum values with context support
func (ac *AggregateClient) MinMaxFieldWithContext(ctx context.Context, field string, qb *query.QueryBuilder) (min, max float64, err error) {
	aq := ac.NewQuery().Min(field, field).Max(field, field)
	if qb != nil {
		aq.queryBuilder = qb.Clone()
	}

	result, err := aq.ExecuteWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	// Extract min and max from result - ServiceNow returns nested structure: stats.min.fieldname, stats.max.fieldname
	if result.Stats != nil {
		// Extract minimum
		if minObj, ok := result.Stats["min"].(map[string]interface{}); ok {
			if minVal, ok := minObj[field]; ok {
				min = parseFloatFromInterface(minVal)
			} else {
				return 0, 0, fmt.Errorf("minimum for field '%s' not found in aggregate result", field)
			}
		} else {
			return 0, 0, fmt.Errorf("minimum not found in aggregate result")
		}

		// Extract maximum
		if maxObj, ok := result.Stats["max"].(map[string]interface{}); ok {
			if maxVal, ok := maxObj[field]; ok {
				max = parseFloatFromInterface(maxVal)
			} else {
				return 0, 0, fmt.Errorf("maximum for field '%s' not found in aggregate result", field)
			}
		} else {
			return 0, 0, fmt.Errorf("maximum not found in aggregate result")
		}
	}

	return min, max, nil
}

// Helper functions for type conversion

func parseIntFromInterface(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

func parseFloatFromInterface(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.0
}
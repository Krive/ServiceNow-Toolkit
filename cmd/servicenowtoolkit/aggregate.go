package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/aggregate"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

var aggregateCmd = &cobra.Command{
	Use:   "aggregate",
	Short: "Data aggregation and analysis operations",
	Long:  "Perform aggregate operations like count, sum, avg, min, max on ServiceNow tables",
}

var aggCountCmd = &cobra.Command{
	Use:   "count [table]",
	Short: "Count records in a table",
	Long:  "Count the number of records in a ServiceNow table with optional filtering",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		filter, _ := cmd.Flags().GetString("filter")

		aggClient := client.Aggregate(tableName)

		var result interface{}
		
		if filter != "" {
			qb := parseSimpleFilter(filter)
			result, err = aggClient.CountRecords(qb)
		} else {
			result, err = aggClient.CountRecords(nil)
		}
		if err != nil {
			return fmt.Errorf("failed to count records: %w", err)
		}

		fmt.Printf("Count: %v\n", result)
		return nil
	},
}

var aggQueryCmd = &cobra.Command{
	Use:   "query [table]",
	Short: "Execute complex aggregate query",
	Long:  "Execute a complex aggregate query with grouping, multiple aggregations, and filtering",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		groupBy, _ := cmd.Flags().GetString("group-by")
		count, _ := cmd.Flags().GetString("count")
		sum, _ := cmd.Flags().GetString("sum")
		avg, _ := cmd.Flags().GetString("avg")
		min, _ := cmd.Flags().GetString("min")
		max, _ := cmd.Flags().GetString("max")
		filter, _ := cmd.Flags().GetString("filter")
		having, _ := cmd.Flags().GetString("having")
		orderBy, _ := cmd.Flags().GetString("order-by")
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		aggClient := client.Aggregate(tableName)
		q := aggClient.NewQuery()

		// Add aggregations
		if count != "" {
			if count == "all" || count == "*" {
				q = q.CountAll("count")
			} else {
				q = q.Count(count, "count_"+count)
			}
		}

		if sum != "" {
			parts := strings.Split(sum, ":")
			field := parts[0]
			alias := field + "_sum"
			if len(parts) > 1 {
				alias = parts[1]
			}
			q = q.Sum(field, alias)
		}

		if avg != "" {
			parts := strings.Split(avg, ":")
			field := parts[0]
			alias := field + "_avg"
			if len(parts) > 1 {
				alias = parts[1]
			}
			q = q.Avg(field, alias)
		}

		if min != "" {
			parts := strings.Split(min, ":")
			field := parts[0]
			alias := field + "_min"
			if len(parts) > 1 {
				alias = parts[1]
			}
			q = q.Min(field, alias)
		}

		if max != "" {
			parts := strings.Split(max, ":")
			field := parts[0]
			alias := field + "_max"
			if len(parts) > 1 {
				alias = parts[1]
			}
			q = q.Max(field, alias)
		}

		// Add grouping
		if groupBy != "" {
			parts := strings.Split(groupBy, ":")
			field := parts[0]
			alias := field
			if len(parts) > 1 {
				alias = parts[1]
			}
			q = q.GroupByField(field, alias)
		}

		// Add filtering
		if filter != "" {
			q = addFilterToAggregateQuery(q, filter)
		}

		// Add having clause
		if having != "" {
			q = q.Having(having)
		}

		// Add ordering
		if orderBy != "" {
			if strings.HasSuffix(orderBy, " DESC") || strings.HasSuffix(orderBy, " desc") {
				field := strings.TrimSuffix(strings.TrimSuffix(orderBy, " DESC"), " desc")
				q = q.OrderByDesc(field)
			} else {
				q = q.OrderByAsc(orderBy)
			}
		}

		// Add limit
		if limit > 0 {
			q = q.Limit(limit)
		}

		// Execute query
		result, err := q.Execute()
		if err != nil {
			return fmt.Errorf("failed to execute aggregate query: %w", err)
		}

		return outputAggregateResult(result, format)
	},
}

var aggStatsCmd = &cobra.Command{
	Use:   "stats [table] [field]",
	Short: "Get statistical summary of a numeric field",
	Long:  "Get count, sum, average, min, max, and standard deviation for a numeric field",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		field := args[1]
		filter, _ := cmd.Flags().GetString("filter")
		format, _ := cmd.Flags().GetString("format")

		aggClient := client.Aggregate(tableName)
		q := aggClient.NewQuery().
			CountAll("count").
			Sum(field, "sum").
			Avg(field, "avg").
			Min(field, "min").
			Max(field, "max")

		// Add filtering
		if filter != "" {
			q = addFilterToAggregateQuery(q, filter)
		}

		result, err := q.Execute()
		if err != nil {
			return fmt.Errorf("failed to get statistics: %w", err)
		}

		return outputStatistics(result, field, format)
	},
}

var aggGroupCmd = &cobra.Command{
	Use:   "group [table] [field]",
	Short: "Group and count records by field value",
	Long:  "Group records by a field value and show counts for each group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		tableName := args[0]
		field := args[1]
		filter, _ := cmd.Flags().GetString("filter")
		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		aggClient := client.Aggregate(tableName)
		q := aggClient.NewQuery().
			CountAll("count").
			GroupByField(field, field).
			OrderByDesc("count")

		// Add filtering
		if filter != "" {
			q = addFilterToAggregateQuery(q, filter)
		}

		// Add limit
		if limit > 0 {
			q = q.Limit(limit)
		}

		result, err := q.Execute()
		if err != nil {
			return fmt.Errorf("failed to group records: %w", err)
		}

		return outputGroupedResult(result, field, format)
	},
}

// Helper functions
func parseSimpleFilter(filter string) *query.QueryBuilder {
	q := query.New()
	
	// Parse simple filter format: field=value^field2=value2
	conditions := strings.Split(filter, "^")
	
	for i, condition := range conditions {
		parts := strings.SplitN(condition, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Try to parse as number or boolean
		if intVal, err := strconv.Atoi(value); err == nil {
			if i > 0 {
				q = q.And()
			}
			q = q.Equals(field, intVal)
		} else if boolVal, err := strconv.ParseBool(value); err == nil {
			if i > 0 {
				q = q.And()
			}
			q = q.Equals(field, boolVal)
		} else {
			if i > 0 {
				q = q.And()
			}
			q = q.Equals(field, value)
		}
	}
	
	return q
}

func addFilterToAggregateQuery(q *aggregate.AggregateQuery, filter string) *aggregate.AggregateQuery {
	// Parse simple filter format: field=value^field2=value2
	conditions := strings.Split(filter, "^")
	
	for i, condition := range conditions {
		parts := strings.SplitN(condition, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if i > 0 {
			q = q.And()
		}
		q = q.Equals(field, value)
	}
	
	return q
}

func outputAggregateResult(result interface{}, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		// Handle the result based on its actual type
		// This would need to be implemented based on the actual aggregate result structure
		fmt.Printf("Aggregate Result:\n")
		if data, err := json.MarshalIndent(result, "", "  "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Printf("%+v\n", result)
		}
		return nil
	}
}

func outputStatistics(result interface{}, field, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		fmt.Printf("Statistics for field '%s':\n", field)
		if data, err := json.MarshalIndent(result, "", "  "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Printf("%+v\n", result)
		}
		return nil
	}
}

func outputGroupedResult(result interface{}, field, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(result)
	default:
		fmt.Printf("Grouped by '%s':\n", field)
		if data, err := json.MarshalIndent(result, "", "  "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Printf("%+v\n", result)
		}
		return nil
	}
}

func init() {
	// Count command flags
	aggCountCmd.Flags().StringP("filter", "f", "", "Filter criteria (field=value^field2=value2)")

	// Query command flags
	aggQueryCmd.Flags().StringP("group-by", "g", "", "Group by field (field or field:alias)")
	aggQueryCmd.Flags().StringP("count", "c", "", "Count field (* for all records)")
	aggQueryCmd.Flags().StringP("sum", "s", "", "Sum field (field or field:alias)")
	aggQueryCmd.Flags().StringP("avg", "a", "", "Average field (field or field:alias)")
	aggQueryCmd.Flags().StringP("min", "m", "", "Minimum field (field or field:alias)")
	aggQueryCmd.Flags().StringP("max", "M", "", "Maximum field (field or field:alias)")
	aggQueryCmd.Flags().StringP("filter", "f", "", "Filter criteria")
	aggQueryCmd.Flags().StringP("having", "H", "", "Having clause")
	aggQueryCmd.Flags().StringP("order-by", "o", "", "Order by field (append ' DESC' for descending)")
	aggQueryCmd.Flags().IntP("limit", "l", 0, "Limit number of results")
	aggQueryCmd.Flags().StringP("format", "", "table", "Output format (table, json)")

	// Stats command flags
	aggStatsCmd.Flags().StringP("filter", "f", "", "Filter criteria")
	aggStatsCmd.Flags().StringP("format", "", "table", "Output format (table, json)")

	// Group command flags
	aggGroupCmd.Flags().StringP("filter", "f", "", "Filter criteria")
	aggGroupCmd.Flags().IntP("limit", "l", 10, "Limit number of groups")
	aggGroupCmd.Flags().StringP("format", "", "table", "Output format (table, json)")

	// Add subcommands
	aggregateCmd.AddCommand(aggCountCmd, aggQueryCmd, aggStatsCmd, aggGroupCmd)
	rootCmd.AddCommand(aggregateCmd)
}
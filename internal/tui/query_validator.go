package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/query"
)

// ValidationError represents a query validation error
type ValidationError struct {
	Field       string `json:"field,omitempty"`
	Operator    string `json:"operator,omitempty"`
	Value       string `json:"value,omitempty"`
	Message     string `json:"message"`
	Severity    ValidationSeverity `json:"severity"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ValidationSeverity indicates the severity of a validation issue
type ValidationSeverity string

const (
	SeverityError     ValidationSeverity = "error"
	SeverityWarning   ValidationSeverity = "warning"
	SeverityInfo      ValidationSeverity = "info"
)

// ValidationResult contains the results of query validation
type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors"`
	Query   string           `json:"query"`
}

// QueryValidator handles validation of ServiceNow queries
type QueryValidator struct {
	tableMetadata *TableFieldMetadata
}

// NewQueryValidator creates a new query validator
func NewQueryValidator(tableMetadata *TableFieldMetadata) *QueryValidator {
	return &QueryValidator{
		tableMetadata: tableMetadata,
	}
}

// ValidateCondition validates a single query condition
func (v *QueryValidator) ValidateCondition(condition QueryCondition) []ValidationError {
	var errors []ValidationError

	// Field validation
	if condition.Field.Name == "" {
		errors = append(errors, ValidationError{
			Message:  "Field is required",
			Severity: SeverityError,
		})
		return errors
	}

	// Check if field exists in table metadata
	fieldExists := false
	if v.tableMetadata != nil {
		for _, field := range v.tableMetadata.Fields {
			if field.Name == condition.Field.Name {
				fieldExists = true
				break
			}
		}
	}

	if !fieldExists && v.tableMetadata != nil {
		errors = append(errors, ValidationError{
			Field:      condition.Field.Name,
			Message:    fmt.Sprintf("Field '%s' does not exist in table", condition.Field.Name),
			Severity:   SeverityWarning,
			Suggestion: "Use field search to find valid fields",
		})
	}

	// Operator validation
	if condition.Operator == "" {
		errors = append(errors, ValidationError{
			Field:    condition.Field.Name,
			Message:  "Operator is required",
			Severity: SeverityError,
		})
	} else {
		// Validate operator compatibility with field type
		errors = append(errors, v.validateOperatorForFieldType(condition.Field, condition.Operator)...)
	}

	// Value validation
	errors = append(errors, v.validateValue(condition)...)

	return errors
}

// ValidateQuery validates a complete query built from conditions
func (v *QueryValidator) ValidateQuery(conditions []QueryCondition) ValidationResult {
	var allErrors []ValidationError
	
	if len(conditions) == 0 {
		allErrors = append(allErrors, ValidationError{
			Message:  "Query must have at least one condition",
			Severity: SeverityError,
		})
		return ValidationResult{
			IsValid: false,
			Errors:  allErrors,
			Query:   "",
		}
	}
	
	// Validate each condition
	for i, condition := range conditions {
		conditionErrors := v.ValidateCondition(condition)
		for _, err := range conditionErrors {
			err.Field = fmt.Sprintf("Condition %d: %s", i+1, err.Field)
			allErrors = append(allErrors, err)
		}
	}
	
	// Build query string for validation
	queryStr := v.buildQueryString(conditions)
	
	// Validate query syntax
	syntaxErrors := v.validateQuerySyntax(queryStr)
	allErrors = append(allErrors, syntaxErrors...)
	
	// Check for potential performance issues
	performanceWarnings := v.checkPerformanceIssues(conditions, queryStr)
	allErrors = append(allErrors, performanceWarnings...)
	
	// Determine if query is valid (only errors, not warnings)
	isValid := true
	for _, err := range allErrors {
		if err.Severity == SeverityError {
			isValid = false
			break
		}
	}
	
	return ValidationResult{
		IsValid: isValid,
		Errors:  allErrors,
		Query:   queryStr,
	}
}

// validateOperatorForFieldType checks if operator is compatible with field type
func (v *QueryValidator) validateOperatorForFieldType(field FieldMetadata, op query.Operator) []ValidationError {
	var errors []ValidationError

	switch field.Type {
	case FieldTypeReference:
		// Reference fields should typically use = or !=, CONTAINS for display values
		if op != query.OpEquals && op != query.OpNotEquals && op != query.OpContains && op != query.OpStartsWith {
			errors = append(errors, ValidationError{
				Field:      field.Name,
				Operator:   string(op),
				Message:    "Reference fields work best with =, !=, CONTAINS, or STARTSWITH operators",
				Severity:   SeverityWarning,
				Suggestion: "Consider using = for sys_id or CONTAINS for display names",
			})
		}

	case FieldTypeDate, FieldTypeDateTime:
		// Date fields should use comparison operators or range operators
		validDateOps := []query.Operator{
			query.OpEquals, query.OpNotEquals, query.OpGreaterThan, query.OpGreaterThanOrEqual,
			query.OpLessThan, query.OpLessThanOrEqual, query.OpBetween, query.OpBefore, query.OpAfter,
		}
		isValid := false
		for _, validOp := range validDateOps {
			if op == validOp {
				isValid = true
				break
			}
		}
		if !isValid {
			errors = append(errors, ValidationError{
				Field:      field.Name,
				Operator:   string(op),
				Message:    "Date fields should use comparison operators (=, !=, >, >=, <, <=) or date operators",
				Severity:   SeverityWarning,
				Suggestion: "Use >= for 'on or after' or <= for 'on or before'",
			})
		}

	case FieldTypeInteger, FieldTypeDecimal:
		// Numeric fields shouldn't use text operators
		if op == query.OpContains || op == query.OpStartsWith || op == query.OpEndsWith {
			errors = append(errors, ValidationError{
				Field:      field.Name,
				Operator:   string(op),
				Message:    "Numeric fields should not use text operators (CONTAINS, STARTSWITH, ENDSWITH)",
				Severity:   SeverityWarning,
				Suggestion: "Use comparison operators (=, !=, >, >=, <, <=) for numbers",
			})
		}

	case FieldTypeBoolean:
		// Boolean fields should only use = or !=
		if op != query.OpEquals && op != query.OpNotEquals {
			errors = append(errors, ValidationError{
				Field:      field.Name,
				Operator:   string(op),
				Message:    "Boolean fields should only use = or != operators",
				Severity:   SeverityWarning,
				Suggestion: "Use = true/false or != true/false",
			})
		}
	}

	return errors
}

// validateValue validates the value for a condition
func (v *QueryValidator) validateValue(condition QueryCondition) []ValidationError {
	var errors []ValidationError

	// Skip validation for operators that don't require values
	operatorInfo := getOperatorInfo(condition.Operator, condition.Field.Type)
	if !operatorInfo.RequiresValue {
		return errors
	}

	if strings.TrimSpace(condition.Value) == "" && condition.StartDate == nil {
		errors = append(errors, ValidationError{
			Field:    condition.Field.Name,
			Operator: string(condition.Operator),
			Message:  "Value is required for this operator",
			Severity: SeverityError,
		})
		return errors
	}

	// Type-specific validation
	switch condition.Field.Type {
	case FieldTypeInteger:
		if condition.Value != "" {
			if _, err := strconv.Atoi(condition.Value); err != nil {
				errors = append(errors, ValidationError{
					Field:      condition.Field.Name,
					Value:      condition.Value,
					Message:    "Value must be a valid integer",
					Severity:   SeverityError,
					Suggestion: "Enter a whole number (e.g., 123)",
				})
			}
		}

	case FieldTypeDecimal:
		if condition.Value != "" {
			if _, err := strconv.ParseFloat(condition.Value, 64); err != nil {
				errors = append(errors, ValidationError{
					Field:      condition.Field.Name,
					Value:      condition.Value,
					Message:    "Value must be a valid decimal number",
					Severity:   SeverityError,
					Suggestion: "Enter a decimal number (e.g., 123.45)",
				})
			}
		}

	case FieldTypeBoolean:
		if condition.Value != "" {
			value := strings.ToLower(strings.TrimSpace(condition.Value))
			if value != "true" && value != "false" && value != "1" && value != "0" {
				errors = append(errors, ValidationError{
					Field:      condition.Field.Name,
					Value:      condition.Value,
					Message:    "Boolean value must be true, false, 1, or 0",
					Severity:   SeverityError,
					Suggestion: "Use 'true', 'false', '1', or '0'",
				})
			}
		}

	case FieldTypeDate, FieldTypeDateTime:
		if condition.Value != "" {
			// Try to parse ServiceNow date format
			if !v.isValidServiceNowDate(condition.Value) {
				errors = append(errors, ValidationError{
					Field:      condition.Field.Name,
					Value:      condition.Value,
					Message:    "Invalid date format",
					Severity:   SeverityError,
					Suggestion: "Use format: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS",
				})
			}
		}

	case FieldTypeChoice:
		// For choice fields, warn if value might not be a valid choice
		if condition.Value != "" && len(condition.Field.Choices) > 0 {
			validChoice := false
			for _, choice := range condition.Field.Choices {
				if choice.Value == condition.Value || choice.Label == condition.Value {
					validChoice = true
					break
				}
			}
			if !validChoice {
				errors = append(errors, ValidationError{
					Field:      condition.Field.Name,
					Value:      condition.Value,
					Message:    "Value may not be a valid choice for this field",
					Severity:   SeverityWarning,
					Suggestion: "Use the choice dropdown to select valid values",
				})
			}
		}

	case FieldTypeString:
		// Check for potentially problematic characters
		if strings.Contains(condition.Value, "^") || strings.Contains(condition.Value, "=") {
			errors = append(errors, ValidationError{
				Field:      condition.Field.Name,
				Value:      condition.Value,
				Message:    "Value contains special characters that may cause issues",
				Severity:   SeverityWarning,
				Suggestion: "Special characters (^ =) in values may need to be escaped",
			})
		}

		// Check for very long strings that might cause performance issues
		if len(condition.Value) > 4000 {
			errors = append(errors, ValidationError{
				Field:      condition.Field.Name,
				Value:      fmt.Sprintf("%.50s...", condition.Value),
				Message:    "Very long values may cause performance issues",
				Severity:   SeverityWarning,
				Suggestion: "Consider using shorter, more specific search terms",
			})
		}
	}

	return errors
}

// validateQuerySyntax validates the built query syntax
func (v *QueryValidator) validateQuerySyntax(queryStr string) []ValidationError {
	var errors []ValidationError

	if queryStr == "" {
		return errors
	}

	// Check for common syntax issues
	
	// Unmatched operators
	if strings.HasPrefix(queryStr, "^") || strings.HasSuffix(queryStr, "^") {
		errors = append(errors, ValidationError{
			Message:  "Query has unmatched logical operators (^)",
			Severity: SeverityWarning,
		})
	}

	// Multiple consecutive operators
	if strings.Contains(queryStr, "^^") {
		errors = append(errors, ValidationError{
			Message:  "Query contains consecutive logical operators",
			Severity: SeverityError,
		})
	}

	// Validate parentheses (if used)
	if strings.Contains(queryStr, "(") || strings.Contains(queryStr, ")") {
		if !v.hasBalancedParentheses(queryStr) {
			errors = append(errors, ValidationError{
				Message:  "Query has unbalanced parentheses",
				Severity: SeverityError,
			})
		}
	}

	// Check for potentially problematic patterns
	if len(queryStr) > 8000 {
		errors = append(errors, ValidationError{
			Message:  "Query is very long and may cause performance issues",
			Severity: SeverityWarning,
			Suggestion: "Consider simplifying the query or breaking it into multiple searches",
		})
	}

	return errors
}

// checkPerformanceIssues identifies potential performance problems
func (v *QueryValidator) checkPerformanceIssues(conditions []QueryCondition, queryStr string) []ValidationError {
	var errors []ValidationError

	// Too many conditions
	if len(conditions) > 20 {
		errors = append(errors, ValidationError{
			Message:  fmt.Sprintf("Query has %d conditions which may be slow", len(conditions)),
			Severity: SeverityWarning,
			Suggestion: "Consider simplifying the query for better performance",
		})
	}

	// Multiple CONTAINS operations
	containsCount := 0
	for _, condition := range conditions {
		if condition.Operator == query.OpContains {
			containsCount++
		}
	}
	if containsCount > 5 {
		errors = append(errors, ValidationError{
			Message:  fmt.Sprintf("Query has %d CONTAINS operations which may be slow", containsCount),
			Severity: SeverityWarning,
			Suggestion: "Consider using more specific operators like STARTSWITH or exact matches",
		})
	}

	// OR operations (which can be slower)
	if strings.Contains(queryStr, "^OR") {
		orCount := strings.Count(queryStr, "^OR")
		if orCount > 3 {
			errors = append(errors, ValidationError{
				Message:  fmt.Sprintf("Query has %d OR operations which may be slow", orCount),
				Severity: SeverityWarning,
				Suggestion: "OR operations can be slow on large tables",
			})
		}
	}

	return errors
}

// Helper methods

func (v *QueryValidator) buildQueryString(conditions []QueryCondition) string {
	if len(conditions) == 0 {
		return ""
	}

	var parts []string
	for i, condition := range conditions {
		part := fmt.Sprintf("%s%s%s", condition.Field.Name, condition.Operator, condition.Value)
		
		if i > 0 && len(conditions) > 1 {
			// Add logical operator from previous condition
			prevCondition := conditions[i-1]
			if prevCondition.LogicalOp == LogicalOr {
				parts[len(parts)-1] = parts[len(parts)-1] + "^OR" + part
			} else {
				parts = append(parts, part)
			}
		} else {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, "^")
}

func (v *QueryValidator) isValidServiceNowDate(dateStr string) bool {
	// Common ServiceNow date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"01-02-2006",
		"01/02/2006",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, dateStr); err == nil {
			return true
		}
	}

	return false
}

func (v *QueryValidator) hasBalancedParentheses(s string) bool {
	count := 0
	for _, char := range s {
		if char == '(' {
			count++
		} else if char == ')' {
			count--
			if count < 0 {
				return false
			}
		}
	}
	return count == 0
}

// ValidateRawQuery validates a raw query string (for manual entry)
func (v *QueryValidator) ValidateRawQuery(rawQuery string) ValidationResult {
	var errors []ValidationError

	if strings.TrimSpace(rawQuery) == "" {
		errors = append(errors, ValidationError{
			Message:  "Query cannot be empty",
			Severity: SeverityError,
		})
		return ValidationResult{IsValid: false, Errors: errors, Query: rawQuery}
	}

	// Basic syntax validation for raw queries
	syntaxErrors := v.validateQuerySyntax(rawQuery)
	errors = append(errors, syntaxErrors...)

	// Check for SQL injection patterns (basic check)
	sqlPatterns := []string{
		"drop table", "delete from", "update set", "insert into",
		"exec(", "execute(", "sp_", "xp_",
	}
	lowerQuery := strings.ToLower(rawQuery)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerQuery, pattern) {
			errors = append(errors, ValidationError{
				Message:  "Query contains potentially dangerous SQL patterns",
				Severity: SeverityError,
			})
			break
		}
	}

	// Determine if valid
	isValid := true
	for _, err := range errors {
		if err.Severity == SeverityError {
			isValid = false
			break
		}
	}

	return ValidationResult{
		IsValid: isValid,
		Errors:  errors,
		Query:   rawQuery,
	}
}

// getOperatorInfo returns operator information for validation
func getOperatorInfo(op query.Operator, fieldType FieldType) OperatorInfo {
	// Default operator info
	info := OperatorInfo{
		Operator:      op,
		RequiresValue: true,
		DateOnly:      false,
	}
	
	// Operators that don't require values
	switch op {
	case query.OpIsEmpty, query.OpIsNotEmpty, query.OpToday, query.OpYesterday,
		 query.OpThisWeek, query.OpLastWeek, query.OpThisMonth, query.OpLastMonth,
		 query.OpThisYear, query.OpLastYear:
		info.RequiresValue = false
	}
	
	// Date operators
	switch op {
	case query.OpToday, query.OpYesterday, query.OpThisWeek, query.OpLastWeek,
		 query.OpThisMonth, query.OpLastMonth, query.OpThisYear, query.OpLastYear,
		 query.OpBetween:
		info.DateOnly = true
	}
	
	return info
}
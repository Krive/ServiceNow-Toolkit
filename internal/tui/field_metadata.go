package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

// FieldType represents the type of a field
type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeInteger   FieldType = "integer"
	FieldTypeDecimal   FieldType = "decimal"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeDateTime  FieldType = "glide_date_time"
	FieldTypeDate      FieldType = "glide_date"
	FieldTypeReference FieldType = "reference"
	FieldTypeChoice    FieldType = "choice"
	FieldTypeJournal   FieldType = "journal"
	FieldTypeHTML      FieldType = "html"
	FieldTypeURL       FieldType = "url"
	FieldTypeEmail     FieldType = "email"
	FieldTypePassword  FieldType = "password"
	FieldTypeTranslatedText FieldType = "translated_text"
)

// FieldMetadata represents metadata about a table field
type FieldMetadata struct {
	Name           string                 `json:"name"`
	Label          string                 `json:"label"`
	Type           FieldType              `json:"type"`
	MaxLength      int                    `json:"max_length"`
	Mandatory      bool                   `json:"mandatory"`
	ReadOnly       bool                   `json:"read_only"`
	Reference      string                 `json:"reference,omitempty"`
	Choices        []FieldChoice          `json:"choices,omitempty"`
	DefaultValue   string                 `json:"default_value,omitempty"`
	Description    string                 `json:"description,omitempty"`
	DependentField string                 `json:"dependent_field,omitempty"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
}

// FieldChoice represents a choice option for choice fields
type FieldChoice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// TableFieldMetadata contains all field metadata for a table
type TableFieldMetadata struct {
	TableName string          `json:"table_name"`
	Fields    []FieldMetadata `json:"fields"`
	LoadedAt  time.Time       `json:"loaded_at"`
}

// FieldMetadataService handles loading and caching field metadata
type FieldMetadataService struct {
	client *servicenow.Client
	cache  map[string]*TableFieldMetadata
}

// NewFieldMetadataService creates a new field metadata service
func NewFieldMetadataService(client *servicenow.Client) *FieldMetadataService {
	return &FieldMetadataService{
		client: client,
		cache:  make(map[string]*TableFieldMetadata),
	}
}

// GetFieldMetadata returns field metadata for a table, loading it if necessary
func (fms *FieldMetadataService) GetFieldMetadata(tableName string) (*TableFieldMetadata, error) {
	// Check cache first
	if metadata, exists := fms.cache[tableName]; exists {
		// Cache for 1 hour
		if time.Since(metadata.LoadedAt) < time.Hour {
			return metadata, nil
		}
	}

	// Load from ServiceNow
	metadata, err := fms.loadFieldMetadata(tableName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	fms.cache[tableName] = metadata
	return metadata, nil
}

// loadFieldMetadata loads field metadata from ServiceNow
func (fms *FieldMetadataService) loadFieldMetadata(tableName string) (*TableFieldMetadata, error) {
	// Try multiple approaches to get field metadata
	var records []map[string]interface{}
	var err error
	
	// Approach 1: Standard query by table name
	params := map[string]string{
		"sysparm_query":  fmt.Sprintf("name=%s", tableName),
		"sysparm_fields": "element,column_label,internal_type,max_length,mandatory,read_only,reference,default_value,comments,dependent",
		"sysparm_limit":  "1000",
	}

	records, err = fms.client.Table("sys_dictionary").List(params)
	// TODO: Remove debug output or make it conditional via env var
	// fmt.Printf("DEBUG: Standard query for table '%s': %s\n", tableName, params["sysparm_query"])
	// fmt.Printf("DEBUG: Standard query found %d records\n", len(records))
	
	// Approach 2: If no records, try with table name in name field
	if len(records) == 0 {
		altParams := map[string]string{
			"sysparm_query":  fmt.Sprintf("name.name=%s", tableName),
			"sysparm_fields": "element,column_label,internal_type,max_length,mandatory,read_only,reference,default_value,comments,dependent",
			"sysparm_limit":  "1000",
		}
		
		if altRecords, altErr := fms.client.Table("sys_dictionary").List(altParams); altErr == nil {
			records = altRecords
			// fmt.Printf("DEBUG: Alternative query (name.name) found %d records\n", len(records))
		} else {
			// fmt.Printf("DEBUG: Alternative query failed: %v\n", altErr)
		}
	}
	
	// Approach 3: If still no records, try broader query including parent tables
	if len(records) == 0 {
		extendedParams := map[string]string{
			"sysparm_query":  fmt.Sprintf("name=%s^ORnameSTARTSWITH%s", tableName, tableName),
			"sysparm_fields": "element,column_label,internal_type,max_length,mandatory,read_only,reference,name",
			"sysparm_limit":  "1000",
		}
		
		if extendedRecords, extErr := fms.client.Table("sys_dictionary").List(extendedParams); extErr == nil {
			records = extendedRecords
			// fmt.Printf("DEBUG: Extended query found %d records\n", len(records))
		} else {
			// fmt.Printf("DEBUG: Extended query failed: %v\n", extErr)
		}
	}
	
	// Approach 4: If all else fails, try a simple query to see if sys_dictionary is accessible
	if len(records) == 0 {
		testParams := map[string]string{
			"sysparm_limit":  "5",
			"sysparm_fields": "element,internal_type,name",
		}
		
		if _, testErr := fms.client.Table("sys_dictionary").List(testParams); testErr == nil {
			// fmt.Printf("DEBUG: Test query succeeded - sys_dictionary is accessible, found %d records\n", len(testRecords))
			// if len(testRecords) > 0 {
			//     fmt.Printf("DEBUG: Sample record fields: %v\n", testRecords[0])
			// }
		} else {
			// fmt.Printf("DEBUG: Test query failed - sys_dictionary may not be accessible: %v\n", testErr)
			err = testErr
		}
	}
	
	if err != nil && len(records) == 0 {
		return nil, fmt.Errorf("failed to load field metadata from sys_dictionary: %w", err)
	}

	metadata := &TableFieldMetadata{
		TableName: tableName,
		Fields:    make([]FieldMetadata, 0),
		LoadedAt:  time.Now(),
	}

	// Process each field record
	for _, record := range records {
		fieldName := getString(record, "element")
		internalType := getString(record, "internal_type")
		
		// Debug: Log field details for first few fields
		// if i < 5 {
		//     fmt.Printf("DEBUG: Field %d - element='%s', internal_type='%s', reference='%s'\n", 
		//         i, fieldName, internalType, getString(record, "reference"))
		// }
		
		field := FieldMetadata{
			Name:        fieldName,
			Label:       getString(record, "column_label"),
			Type:        mapServiceNowFieldType(internalType),
			MaxLength:   getInt(record, "max_length"),
			Mandatory:   getBool(record, "mandatory"),
			ReadOnly:    getBool(record, "read_only"),
			Reference:   getString(record, "reference"),
			DefaultValue: getString(record, "default_value"),
			Description: getString(record, "comments"),
			DependentField: getString(record, "dependent"),
			Attributes:  make(map[string]interface{}),
		}

		// Load choices for choice fields
		if field.Type == FieldTypeChoice && field.Name != "" {
			choices, err := fms.loadFieldChoices(tableName, field.Name)
			if err == nil {
				field.Choices = choices
			}
		}

		metadata.Fields = append(metadata.Fields, field)
	}

	// If we still have no fields after processing, the table might not exist or user might not have access
	if len(metadata.Fields) == 0 {
		return nil, fmt.Errorf("no field metadata found for table '%s' - table may not exist or user may not have read access to sys_dictionary", tableName)
	}

	return metadata, nil
}

// loadFieldChoices loads choice options for a choice field
func (fms *FieldMetadataService) loadFieldChoices(tableName, fieldName string) ([]FieldChoice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params := map[string]string{
		"sysparm_query":  fmt.Sprintf("name=%s^element=%s", tableName, fieldName),
		"sysparm_fields": "value,label",
		"sysparm_limit":  "100",
	}

	records, err := fms.client.Table("sys_choice").ListWithContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to load field choices: %w", err)
	}

	choices := make([]FieldChoice, 0, len(records))
	for _, record := range records {
		choice := FieldChoice{
			Value: getString(record, "value"),
			Label: getString(record, "label"),
		}
		if choice.Label == "" {
			choice.Label = choice.Value
		}
		choices = append(choices, choice)
	}

	return choices, nil
}

// GetFieldsByType returns fields of a specific type
func (metadata *TableFieldMetadata) GetFieldsByType(fieldType FieldType) []FieldMetadata {
	var fields []FieldMetadata
	for _, field := range metadata.Fields {
		if field.Type == fieldType {
			fields = append(fields, field)
		}
	}
	return fields
}

// GetDateTimeFields returns all datetime and date fields
func (metadata *TableFieldMetadata) GetDateTimeFields() []FieldMetadata {
	var fields []FieldMetadata
	for _, field := range metadata.Fields {
		if field.Type == FieldTypeDateTime || field.Type == FieldTypeDate {
			fields = append(fields, field)
		}
	}
	return fields
}

// GetChoiceFields returns all choice fields
func (metadata *TableFieldMetadata) GetChoiceFields() []FieldMetadata {
	return metadata.GetFieldsByType(FieldTypeChoice)
}

// GetReferenceFields returns all reference fields
func (metadata *TableFieldMetadata) GetReferenceFields() []FieldMetadata {
	return metadata.GetFieldsByType(FieldTypeReference)
}

// GetSearchableFields returns fields that are good for text search
func (metadata *TableFieldMetadata) GetSearchableFields() []FieldMetadata {
	var fields []FieldMetadata
	for _, field := range metadata.Fields {
		switch field.Type {
		case FieldTypeString, FieldTypeJournal, FieldTypeHTML, FieldTypeTranslatedText:
			// Include text-based fields that are not read-only and have reasonable length
			if !field.ReadOnly && field.MaxLength > 10 {
				fields = append(fields, field)
			}
		}
	}
	return fields
}

// FindField finds a field by name
func (metadata *TableFieldMetadata) FindField(name string) *FieldMetadata {
	for _, field := range metadata.Fields {
		if field.Name == name {
			return &field
		}
	}
	return nil
}

// GetDisplayFields returns fields that are commonly displayed in lists
func (metadata *TableFieldMetadata) GetDisplayFields() []FieldMetadata {
	// Priority order for display fields
	priorityFields := []string{
		"number", "sys_id", "short_description", "description", "state", "priority",
		"assigned_to", "sys_created_on", "sys_updated_on", "sys_created_by",
		"active", "name", "title", "subject", "category", "subcategory",
	}

	var fields []FieldMetadata
	fieldMap := make(map[string]FieldMetadata)

	// Create a map for quick lookup
	for _, field := range metadata.Fields {
		fieldMap[field.Name] = field
	}

	// Add priority fields first
	for _, fieldName := range priorityFields {
		if field, exists := fieldMap[fieldName]; exists {
			fields = append(fields, field)
			delete(fieldMap, fieldName) // Remove from map to avoid duplicates
		}
	}

	// Add remaining important fields (not read-only, reasonable length)
	for _, field := range fieldMap {
		if len(fields) >= 12 { // Limit to prevent UI overflow
			break
		}
		
		if !field.ReadOnly && field.Type != FieldTypePassword && 
		   field.Type != FieldTypeJournal && field.MaxLength < 1000 {
			fields = append(fields, field)
		}
	}

	return fields
}

// mapServiceNowFieldType maps ServiceNow internal_type to our FieldType
func mapServiceNowFieldType(snType string) FieldType {
	switch snType {
	case "glide_date_time":
		return FieldTypeDateTime
	case "glide_date":
		return FieldTypeDate
	case "reference":
		return FieldTypeReference
	case "choice":
		return FieldTypeChoice
	case "boolean":
		return FieldTypeBoolean
	case "integer":
		return FieldTypeInteger
	case "decimal", "float", "numeric":
		return FieldTypeDecimal
	case "journal", "journal_input":
		return FieldTypeJournal
	case "html":
		return FieldTypeHTML
	case "url":
		return FieldTypeURL
	case "email":
		return FieldTypeEmail
	case "password", "password2":
		return FieldTypePassword
	case "translated_text":
		return FieldTypeTranslatedText
	case "string", "syslog":
		return FieldTypeString
	default:
		// Default to string for unknown types
		return FieldTypeString
	}
}

// Helper functions for extracting values from ServiceNow records
func getString(record map[string]interface{}, key string) string {
	if val, exists := record[key]; exists && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getInt(record map[string]interface{}, key string) int {
	if val, exists := record[key]; exists && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if strings.TrimSpace(v) == "" {
				return 0
			}
			// Try to parse string as int
			return 0
		}
	}
	return 0
}

func getBool(record map[string]interface{}, key string) bool {
	if val, exists := record[key]; exists && val != nil {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return strings.ToLower(v) == "true" || v == "1"
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	return false
}
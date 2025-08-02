package tui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SavedFilter represents a saved filter
type SavedFilter struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	TableName   string           `json:"table_name"`
	Query       string           `json:"query"`
	Conditions  []QueryCondition `json:"conditions"`
	CreatedAt   time.Time        `json:"created_at"`
	LastUsed    time.Time        `json:"last_used"`
	UseCount    int              `json:"use_count"`
	IsFavorite  bool             `json:"is_favorite"`
	Tags        []string         `json:"tags,omitempty"`
}

// FilterHistory represents the filter usage history
type FilterHistory struct {
	TableName  string           `json:"table_name"`
	Query      string           `json:"query"`
	Conditions []QueryCondition `json:"conditions"`
	Timestamp  time.Time        `json:"timestamp"`
	Duration   time.Duration    `json:"duration,omitempty"`
	Results    int              `json:"results,omitempty"`
}

// FilterManager handles saving, loading, and managing filters
type FilterManager struct {
	configDir    string
	filtersFile  string
	historyFile  string
	filters      map[string]*SavedFilter
	history      []*FilterHistory
	maxHistory   int
}

// NewFilterManager creates a new filter manager
func NewFilterManager() (*FilterManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".servicenowtoolkit")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	fm := &FilterManager{
		configDir:   configDir,
		filtersFile: filepath.Join(configDir, "saved_filters.json"),
		historyFile: filepath.Join(configDir, "filter_history.json"),
		filters:     make(map[string]*SavedFilter),
		history:     make([]*FilterHistory, 0),
		maxHistory:  100, // Keep last 100 history entries
	}
	
	// Load existing data
	fm.loadFilters()
	fm.loadHistory()
	
	return fm, nil
}

// SaveFilter saves a filter with the given name
func (fm *FilterManager) SaveFilter(name, description, tableName, query string, conditions []QueryCondition, isFavorite bool) (*SavedFilter, error) {
	filter := &SavedFilter{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:        name,
		Description: description,
		TableName:   tableName,
		Query:       query,
		Conditions:  conditions,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UseCount:    0,
		IsFavorite:  isFavorite,
		Tags:        make([]string, 0),
	}
	
	fm.filters[filter.ID] = filter
	
	if err := fm.saveFilters(); err != nil {
		return nil, err
	}
	
	return filter, nil
}

// UpdateFilter updates an existing filter
func (fm *FilterManager) UpdateFilter(id, name, description string, isFavorite bool, tags []string) error {
	filter, exists := fm.filters[id]
	if !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}
	
	filter.Name = name
	filter.Description = description
	filter.IsFavorite = isFavorite
	filter.Tags = tags
	
	return fm.saveFilters()
}

// DeleteFilter deletes a filter
func (fm *FilterManager) DeleteFilter(id string) error {
	if _, exists := fm.filters[id]; !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}
	
	delete(fm.filters, id)
	return fm.saveFilters()
}

// GetFilter retrieves a filter by ID
func (fm *FilterManager) GetFilter(id string) (*SavedFilter, bool) {
	filter, exists := fm.filters[id]
	return filter, exists
}

// GetFiltersForTable returns all filters for a specific table
func (fm *FilterManager) GetFiltersForTable(tableName string) []*SavedFilter {
	var filters []*SavedFilter
	for _, filter := range fm.filters {
		if filter.TableName == tableName {
			filters = append(filters, filter)
		}
	}
	
	// Sort by last used, then by use count
	sort.Slice(filters, func(i, j int) bool {
		if filters[i].LastUsed != filters[j].LastUsed {
			return filters[i].LastUsed.After(filters[j].LastUsed)
		}
		return filters[i].UseCount > filters[j].UseCount
	})
	
	return filters
}

// GetFavoriteFilters returns all favorite filters
func (fm *FilterManager) GetFavoriteFilters() []*SavedFilter {
	var favorites []*SavedFilter
	for _, filter := range fm.filters {
		if filter.IsFavorite {
			favorites = append(favorites, filter)
		}
	}
	
	// Sort favorites by last used
	sort.Slice(favorites, func(i, j int) bool {
		return favorites[i].LastUsed.After(favorites[j].LastUsed)
	})
	
	return favorites
}

// GetRecentFilters returns recently used filters
func (fm *FilterManager) GetRecentFilters(limit int) []*SavedFilter {
	var recent []*SavedFilter
	for _, filter := range fm.filters {
		recent = append(recent, filter)
	}
	
	// Sort by last used
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].LastUsed.After(recent[j].LastUsed)
	})
	
	if limit > 0 && len(recent) > limit {
		recent = recent[:limit]
	}
	
	return recent
}

// SearchFilters searches filters by name, description, or tags
func (fm *FilterManager) SearchFilters(query string) []*SavedFilter {
	var matches []*SavedFilter
	queryLower := strings.ToLower(query)
	
	for _, filter := range fm.filters {
		// Check name
		if strings.Contains(strings.ToLower(filter.Name), queryLower) {
			matches = append(matches, filter)
			continue
		}
		
		// Check description
		if strings.Contains(strings.ToLower(filter.Description), queryLower) {
			matches = append(matches, filter)
			continue
		}
		
		// Check tags
		for _, tag := range filter.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				matches = append(matches, filter)
				break
			}
		}
	}
	
	return matches
}

// UseFilter records usage of a filter
func (fm *FilterManager) UseFilter(id string) error {
	filter, exists := fm.filters[id]
	if !exists {
		return fmt.Errorf("filter with ID %s not found", id)
	}
	
	filter.LastUsed = time.Now()
	filter.UseCount++
	
	return fm.saveFilters()
}

// AddToHistory adds a filter execution to history
func (fm *FilterManager) AddToHistory(tableName, query string, conditions []QueryCondition, duration time.Duration, results int) {
	historyEntry := &FilterHistory{
		TableName:  tableName,
		Query:      query,
		Conditions: conditions,
		Timestamp:  time.Now(),
		Duration:   duration,
		Results:    results,
	}
	
	// Add to beginning of history
	fm.history = append([]*FilterHistory{historyEntry}, fm.history...)
	
	// Trim history to max size
	if len(fm.history) > fm.maxHistory {
		fm.history = fm.history[:fm.maxHistory]
	}
	
	// Save history
	fm.saveHistory()
}

// GetHistory returns filter history, optionally filtered by table
func (fm *FilterManager) GetHistory(tableName string, limit int) []*FilterHistory {
	var history []*FilterHistory
	
	for _, entry := range fm.history {
		if tableName == "" || entry.TableName == tableName {
			history = append(history, entry)
		}
	}
	
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}
	
	return history
}

// GetPopularQueries returns the most frequently used queries
func (fm *FilterManager) GetPopularQueries(tableName string, limit int) []string {
	queryCount := make(map[string]int)
	
	// Count query usage in history
	for _, entry := range fm.history {
		if tableName == "" || entry.TableName == tableName {
			if entry.Query != "" {
				queryCount[entry.Query]++
			}
		}
	}
	
	// Sort by usage count
	type queryUsage struct {
		query string
		count int
	}
	
	var usage []queryUsage
	for query, count := range queryCount {
		usage = append(usage, queryUsage{query: query, count: count})
	}
	
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].count > usage[j].count
	})
	
	// Extract queries
	var queries []string
	for _, u := range usage {
		queries = append(queries, u.query)
		if limit > 0 && len(queries) >= limit {
			break
		}
	}
	
	return queries
}

// ClearHistory clears the filter history
func (fm *FilterManager) ClearHistory() error {
	fm.history = make([]*FilterHistory, 0)
	return fm.saveHistory()
}

// ExportFilters exports filters to a JSON file
func (fm *FilterManager) ExportFilters(filename string) error {
	data, err := json.MarshalIndent(fm.filters, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}
	
	return ioutil.WriteFile(filename, data, 0644)
}

// ImportFilters imports filters from a JSON file
func (fm *FilterManager) ImportFilters(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	var importedFilters map[string]*SavedFilter
	if err := json.Unmarshal(data, &importedFilters); err != nil {
		return fmt.Errorf("failed to unmarshal filters: %w", err)
	}
	
	// Merge with existing filters, updating IDs to avoid conflicts
	for _, filter := range importedFilters {
		// Generate new ID to avoid conflicts
		filter.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		fm.filters[filter.ID] = filter
	}
	
	return fm.saveFilters()
}

// saveFilters saves filters to disk
func (fm *FilterManager) saveFilters() error {
	data, err := json.MarshalIndent(fm.filters, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}
	
	return ioutil.WriteFile(fm.filtersFile, data, 0644)
}

// loadFilters loads filters from disk
func (fm *FilterManager) loadFilters() error {
	if _, err := os.Stat(fm.filtersFile); os.IsNotExist(err) {
		return nil // File doesn't exist yet, that's OK
	}
	
	data, err := ioutil.ReadFile(fm.filtersFile)
	if err != nil {
		return fmt.Errorf("failed to read filters file: %w", err)
	}
	
	if err := json.Unmarshal(data, &fm.filters); err != nil {
		return fmt.Errorf("failed to unmarshal filters: %w", err)
	}
	
	return nil
}

// saveHistory saves history to disk
func (fm *FilterManager) saveHistory() error {
	data, err := json.MarshalIndent(fm.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}
	
	return ioutil.WriteFile(fm.historyFile, data, 0644)
}

// loadHistory loads history from disk
func (fm *FilterManager) loadHistory() error {
	if _, err := os.Stat(fm.historyFile); os.IsNotExist(err) {
		return nil // File doesn't exist yet, that's OK
	}
	
	data, err := ioutil.ReadFile(fm.historyFile)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}
	
	if err := json.Unmarshal(data, &fm.history); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}
	
	return nil
}

// GetStats returns usage statistics
func (fm *FilterManager) GetStats() map[string]interface{} {
	totalFilters := len(fm.filters)
	favorites := 0
	totalUseCount := 0
	
	for _, filter := range fm.filters {
		if filter.IsFavorite {
			favorites++
		}
		totalUseCount += filter.UseCount
	}
	
	tableUsage := make(map[string]int)
	for _, entry := range fm.history {
		tableUsage[entry.TableName]++
	}
	
	return map[string]interface{}{
		"total_filters":    totalFilters,
		"favorite_filters": favorites,
		"total_use_count":  totalUseCount,
		"history_entries":  len(fm.history),
		"table_usage":      tableUsage,
	}
}
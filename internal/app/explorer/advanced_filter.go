package explorer

import (
	"github.com/Krive/ServiceNow-Toolkit/internal/tui"
)

// initializeAdvancedFiltering initializes the advanced filtering components if not already done
func (m *Model) initializeAdvancedFiltering() {
	if m.client == nil {
		return // Can't initialize without client
	}

	// Initialize field metadata service
	if m.fieldMetadataService == nil {
		m.fieldMetadataService = tui.NewFieldMetadataService(m.client)
	}

	// Initialize filter manager
	if m.filterManager == nil {
		if fm, err := tui.NewFilterManager(); err == nil {
			m.filterManager = fm
		}
	}

	// Initialize filter browser
	if m.filterBrowser == nil && m.filterManager != nil {
		m.filterBrowser = tui.NewFilterBrowserModel(m.filterManager)
	}
}

// initializeQueryBuilderForTable initializes the query builder for the current table
func (m *Model) initializeQueryBuilderForTable() {
	if m.fieldMetadataService == nil || m.currentTable == "" {
		return
	}

	// Load field metadata for current table
	if metadata, err := m.fieldMetadataService.GetFieldMetadata(m.currentTable); err == nil {
		m.tableMetadata = metadata
		m.queryBuilder = tui.NewQueryBuilderModel(m.client, metadata)
	}
}
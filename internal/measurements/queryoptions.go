package measurements

import "time"

type Filter struct {
	Types      []string
	Sources    []string
	From       *time.Time
	To         *time.Time
	Datasource string
}

type QueryOptions struct {
	Filter Filter
	Sort   []Sort
	Limit  int
	Offset int
}

type Sort struct {
	Field     SortField
	Direction SortDirection
}

type SortField string

const (
	SortByTimestamp SortField = "timestamp"
	SortByType      SortField = "type"
	SortByValue     SortField = "value"
	SortByUnit      SortField = "unit"
	SortBySource    SortField = "source"
)

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

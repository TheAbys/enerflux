package metrics

type Filter struct {
	Key string
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

package models

// Namespace puts all tables names under a common
// namespace. This is useful if you want to use
// the same database for several services and don't
// want table names to collide.
var Namespace string

// tableName returns the name of a model's table
// in the database. It uses the namespace to isolate
// the model when this is set in the configuration.
func tableName(modelName string) string {
	if Namespace != "" {
		return Namespace + "_" + modelName
	}
	return modelName
}

type Pagination struct {
	Page    uint64
	PerPage uint64
	Count   uint64
}

func (p *Pagination) Offset() uint64 {
	return (p.Page - 1) * p.PerPage
}

type SortDirection string

const Ascending SortDirection = "asc"
const Descending SortDirection = "desc"

type SortParams struct {
	Fields []SortField
}

type SortField struct {
	Name string
	Dir  SortDirection
}

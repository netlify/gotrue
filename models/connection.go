package models

import (
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
)

type Pagination struct {
	Page    uint64
	PerPage uint64
	Count   uint64
}

func (p *Pagination) Offset() uint64 {
	return (p.Page - 1) * p.PerPage
}

type SortDirection string

const Ascending SortDirection = "ASC"
const Descending SortDirection = "DESC"
const CreatedAt = "created_at"

type SortParams struct {
	Fields []SortField
}

type SortField struct {
	Name string
	Dir  SortDirection
}

func TruncateAll(database *tigris.Database) error {
	ctx := context.TODO()
	if _, err := tigris.GetCollection[User](database).DeleteAll(ctx); err != nil {
		return err
	}
	if _, err := tigris.GetCollection[RefreshToken](database).DeleteAll(ctx); err != nil {
		return err
	}
	if _, err := tigris.GetCollection[AuditLogEntry](database).DeleteAll(ctx); err != nil {
		return err
	}
	if _, err := tigris.GetCollection[Instance](database).DeleteAll(ctx); err != nil {
		return err
	}

	return nil
}

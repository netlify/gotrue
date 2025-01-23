package models

import (
	"github.com/gobuffalo/pop/v5"
	"github.com/netlify/gotrue/storage"
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

func TruncateAll(conn *storage.Connection) error {
	return conn.Transaction(func(tx *storage.Connection) error {

		tables := []pop.Value{User{}, RefreshToken{}, AuditLogEntry{}, Instance{}}
		for _, v := range tables {
			if err := tx.RawQuery("TRUNCATE " + (&pop.Model{Value: v}).TableName()).Exec(); err != nil {
				return err
			}
		}

		return nil
	})
}

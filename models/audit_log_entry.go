package models

import (
	"time"

	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/netlify/gotrue/storage/namespace"
	"github.com/pkg/errors"
	"github.com/tigrisdata/tigris-client-go/filter"
	"github.com/tigrisdata/tigris-client-go/tigris"
)

type AuditAction string
type auditLogType string

const (
	LoginAction                 AuditAction = "login"
	LogoutAction                AuditAction = "logout"
	InviteAcceptedAction        AuditAction = "invite_accepted"
	UserSignedUpAction          AuditAction = "user_signedup"
	UserInvitedAction           AuditAction = "user_invited"
	UserDeletedAction           AuditAction = "user_deleted"
	UserModifiedAction          AuditAction = "user_modified"
	UserRecoveryRequestedAction AuditAction = "user_recovery_requested"
	TokenRevokedAction          AuditAction = "token_revoked"
	TokenRefreshedAction        AuditAction = "token_refreshed"

	account auditLogType = "account"
	team    auditLogType = "team"
	token   auditLogType = "token"
	user    auditLogType = "user"
)

var actionLogTypeMap = map[AuditAction]auditLogType{
	LoginAction:                 account,
	LogoutAction:                account,
	InviteAcceptedAction:        account,
	UserSignedUpAction:          team,
	UserInvitedAction:           team,
	UserDeletedAction:           team,
	TokenRevokedAction:          token,
	TokenRefreshedAction:        token,
	UserModifiedAction:          user,
	UserRecoveryRequestedAction: user,
}

// AuditLogEntry is the database model for audit log entries.
type AuditLogEntry struct {
	ID         uuid.UUID `json:"id" db:"id"  tigris:"primaryKey"`
	InstanceID uuid.UUID `json:"instance_id" db:"instance_id"`
	Payload    JSONMap   `json:"payload" db:"payload"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (AuditLogEntry) TableName() string {
	tableName := "audit_log_entries"

	if namespace.GetNamespace() != "" {
		return namespace.GetNamespace() + "_" + tableName
	}

	return tableName
}

func NewAuditLogEntry(ctx context.Context, database *tigris.Database, instanceID uuid.UUID, actor *User, action AuditAction, traits map[string]interface{}) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return errors.Wrap(err, "Error generating unique id")
	}
	l := AuditLogEntry{
		InstanceID: instanceID,
		ID:         id,
		Payload: JSONMap{
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"actor_id":    actor.ID,
			"actor_email": actor.Email,
			"action":      action,
			"log_type":    actionLogTypeMap[action],
		},
	}

	if name, ok := actor.UserMetaData["full_name"]; ok {
		l.Payload["actor_name"] = name
	}

	if traits != nil {
		l.Payload["traits"] = traits
	}

	_, err = tigris.GetCollection[AuditLogEntry](database).Insert(ctx, &l)
	return errors.Wrap(err, "Database error creating audit log entry")
}

func FindAuditLogEntries(ctx context.Context, database *tigris.Database, instanceID uuid.UUID, filterColumns []string, filterValue string, pageParams *Pagination) ([]*AuditLogEntry, error) {
	// ToDo: Order("created_at desc")
	it, err := tigris.GetCollection[AuditLogEntry](database).Read(ctx, filter.Eq("instance_id", instanceID))
	if err != nil {
		return nil, errors.Wrap(err, "reading audit log entries failed")
	}

	var logs []*AuditLogEntry
	var entry AuditLogEntry
	for it.Next(&entry) {
		e := entry
		if len(filterColumns) == 0 {
			logs = append(logs, &e)
			continue
		}

		filterValue = strings.ToLower(filterValue)
		for _, col := range filterColumns {
			if val, found := e.Payload[col]; found {
				if conv, ok := val.(string); ok && strings.Contains(strings.ToLower(conv), filterValue) {
					logs = append(logs, &e)
				}
			}
		}
	}

	// ToDo: pagination
	/**
	if pageParams != nil {
		err = q.Paginate(int(pageParams.Page), int(pageParams.PerPage)).All(&logs)
		pageParams.Count = uint64(q.Paginator.TotalEntriesSize)
	} else {
		err = q.All(&logs)
	}*/

	return logs, err
}

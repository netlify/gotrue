package models

import (
	"fmt"
	"time"

	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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

	account auditLogType = "gotrue-account"
	team    auditLogType = "gotrue-team"
	token   auditLogType = "gotrue-token"
	user    auditLogType = "gotrue-user"
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
	InstanceID uuid.UUID `json:"-" db:"instance_id"`
	ID         uuid.UUID `json:"id" db:"id"`

	Payload JSONMap `json:"payload" db:"payload"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func NewAuditLogEntry(tx *storage.Connection, instanceID uuid.UUID, actor *User, action AuditAction, traits map[string]interface{}) error {
	id, err := uuid.NewV4()
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

	return errors.Wrap(tx.Create(&l), "Database error creating audit log entry")
}

func FindAuditLogEntries(tx *storage.Connection, instanceID uuid.UUID, filterColumns []string, filterValue string, pageParams *Pagination) ([]*AuditLogEntry, error) {
	q := tx.Q().Where("instance_id = ?", instanceID)

	if len(filterColumns) > 0 && filterValue != "" {
		lf := "%" + filterValue + "%"
		for _, col := range filterColumns {
			q = q.Where(fmt.Sprintf("payload->>'$.%s' COLLATE utf8mb4_unicode_ci LIKE ?", col), lf)
		}
	}

	logs := []*AuditLogEntry{}
	var err error
	if pageParams != nil {
		err = q.Paginate(int(pageParams.Page), int(pageParams.PerPage)).All(&logs)
		pageParams.Count = uint64(q.Paginator.TotalEntriesSize)
	} else {
		err = q.All(&logs)
	}

	return logs, err
}

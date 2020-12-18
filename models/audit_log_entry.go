package models

import (
	"bytes"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
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

func init() {
	storage.AddMigration(&AuditLogEntry{})
}

// AuditLogEntry is the database model for audit log entries.
type AuditLogEntry struct {
	InstanceID uuid.UUID `json:"-" gorm:"index:audit_logs_instance_id_idx;type:varchar(255) DEFAULT NULL"`
	ID         uuid.UUID `json:"id" gorm:"primaryKey;type:varchar(255) NOT NULL"`

	Payload JSONMap `json:"payload" gorm:"type:JSON NULL DEFAULT NULL"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp NULL DEFAULT NULL"`
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

	return errors.Wrap(tx.Create(&l).Error, "Database error creating audit log entry")
}

func FindAuditLogEntries(tx *storage.Connection, instanceID uuid.UUID, filterColumns []string, filterValue string, pageParams *Pagination) ([]*AuditLogEntry, error) {
	q := tx.Model(AuditLogEntry{}).Order("created_at desc").Where("instance_id = ?", instanceID)

	if len(filterColumns) > 0 && filterValue != "" {
		lf := "%" + filterValue + "%"

		builder := bytes.NewBufferString("(")
		values := make([]interface{}, len(filterColumns))

		for idx, col := range filterColumns {
			builder.WriteString(fmt.Sprintf("payload->>'$.%s' COLLATE utf8mb4_unicode_ci LIKE ?", col))
			values[idx] = lf

			if idx+1 < len(filterColumns) {
				builder.WriteString(" OR ")
			}
		}
		builder.WriteString(")")

		q = q.Where(builder.String(), values...)
	}

	logs := []*AuditLogEntry{}
	var err error
	if pageParams != nil {
		p := paginator.New(adapter.NewGORMAdapter(q), int(pageParams.PerPage))
		p.SetPage(int(pageParams.Page))
		if err = p.Results(&logs); err != nil {
			return nil, err
		}
		var cnt int
		if cnt, err = p.PageNums(); err != nil {
			return nil, err
		}
		pageParams.Count = uint64(cnt)
	} else {
		err = q.Find(&logs).Error
	}

	return logs, err
}

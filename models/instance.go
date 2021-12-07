package models

import (
	"database/sql"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/namespace"
	"github.com/pkg/errors"
)

const baseConfigKey = ""

type Instance struct {
	ID uuid.UUID `json:"id" db:"id"`
	// Netlify UUID
	UUID uuid.UUID `json:"uuid,omitempty" db:"uuid"`

	BaseConfig *conf.Configuration `json:"config" db:"raw_base_config"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (Instance) TableName() string {
	tableName := "instances"

	if namespace.GetNamespace() != "" {
		return namespace.GetNamespace() + "_" + tableName
	}

	return tableName
}

// Config loads the the base configuration values with defaults.
func (i *Instance) Config() (*conf.Configuration, error) {
	if i.BaseConfig == nil {
		return nil, errors.New("no configuration data available")
	}

	baseConf := &conf.Configuration{}
	*baseConf = *i.BaseConfig
	baseConf.ApplyDefaults()

	return baseConf, nil
}

// UpdateConfig updates the base config
func (i *Instance) UpdateConfig(tx *storage.Connection, config *conf.Configuration) error {
	i.BaseConfig = config
	return tx.UpdateOnly(i, "raw_base_config")
}

// GetInstance finds an instance by ID
func GetInstance(tx *storage.Connection, instanceID uuid.UUID) (*Instance, error) {
	instance := Instance{}
	if err := tx.Find(&instance, instanceID); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func GetInstanceByUUID(tx *storage.Connection, uuid uuid.UUID) (*Instance, error) {
	instance := Instance{}
	if err := tx.Where("uuid = ?", uuid).First(&instance); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func DeleteInstance(conn *storage.Connection, instance *Instance) error {
	return conn.Transaction(func(tx *storage.Connection) error {
		delModels := map[string]*pop.Model{
			"user":          &pop.Model{Value: &User{}},
			"refresh token": &pop.Model{Value: &RefreshToken{}},
		}

		for name, dm := range delModels {
			if err := tx.RawQuery("DELETE FROM "+dm.TableName()+" WHERE instance_id = ?", instance.ID).Exec(); err != nil {
				return errors.Wrapf(err, "Error deleting %s records", name)
			}
		}

		return errors.Wrap(tx.Destroy(instance), "Error deleting instance record")
	})
}

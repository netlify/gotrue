package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/namespace"
	"github.com/pkg/errors"
	"context"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"github.com/tigrisdata/tigris-client-go/filter"
	"github.com/tigrisdata/tigris-client-go/fields"
)

const baseConfigKey = ""

type Instance struct {
	ID uuid.UUID `json:"id" db:"id" tigris:"primaryKey"`
	// Netlify UUID
	UUID uuid.UUID `json:"uuid,omitempty" db:"uuid"`

	BaseConfig *conf.Configuration `json:"config" db:"config"`

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

// Config loads the base configuration values with defaults.
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
func (i *Instance) UpdateConfig(ctx context.Context, database *tigris.Database, config *conf.Configuration) error {
	i.BaseConfig = config
	_, err := tigris.GetCollection[Instance](database).Update(ctx, filter.Eq("id", i.ID), fields.Set("config", i.BaseConfig))
	return err
}

// GetInstance finds an instance by ID
func GetInstance(ctx context.Context, database *tigris.Database, instanceID uuid.UUID) (*Instance, error) {
	instance, err := tigris.GetCollection[Instance](database).ReadOne(ctx, filter.Eq("id", instanceID))
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, InstanceNotFoundError{}
	}
	return instance, nil
}

func GetInstanceByUUID(ctx context.Context, database *tigris.Database, uuid uuid.UUID) (*Instance, error) {
	instance, err := tigris.GetCollection[Instance](database).ReadOne(ctx, filter.Eq("uuid", uuid))
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, InstanceNotFoundError{}
	}

	return instance, nil
}

func DeleteInstance(ctx context.Context, database *tigris.Database, instance *Instance) error {
	return database.Tx(ctx, func(ctx context.Context) error {
		_, err := tigris.GetCollection[User](database).Delete(ctx, filter.Eq("instance_id", instance.ID))
		if err != nil {
			return errors.Wrap(err, "Error deleting user record")
		}

		_, err = tigris.GetCollection[RefreshToken](database).Delete(ctx, filter.Eq("instance_id", instance.ID))
		if err != nil {
			return errors.Wrap(err, "Error deleting refresh token record")
		}

		_, err = tigris.GetCollection[Instance](database).Delete(ctx, filter.Eq("id", instance.ID))
		if err != nil {
			return errors.Wrap(err, "Error deleting instance record")
		}

		return nil
	})
}

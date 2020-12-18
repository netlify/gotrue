package models

import (
	"gorm.io/gorm"
	"time"

	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

const baseConfigKey = ""

func init() {
	storage.AddMigration(&Instance{})
}

type Instance struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:varchar(255) NOT NULL"`
	// Netlify UUID
	UUID uuid.UUID `json:"uuid,omitempty" gorm:"type:varchar(255) DEFAULT NULL"`

	BaseConfig *conf.Configuration `json:"config" gorm:"column:raw_base_config"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp NULL DEFAULT NULL"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp NULL DEFAULT NULL"`
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
	return tx.Model(&i).Select("raw_base_config").Updates(i).Error
}

// GetInstance finds an instance by ID
func GetInstance(tx *storage.Connection, instanceID uuid.UUID) (*Instance, error) {
	instance := Instance{}
	if err := tx.Find(&instance, instanceID).Error; err != nil {
		if errors.Cause(err) == gorm.ErrRecordNotFound {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func GetInstanceByUUID(tx *storage.Connection, uuid uuid.UUID) (*Instance, error) {
	instance := Instance{}
	if err := tx.Where("uuid = ?", uuid).First(&instance).Error; err != nil {
		if errors.Cause(err) == gorm.ErrRecordNotFound {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func DeleteInstance(conn *storage.Connection, instance *Instance) error {
	return conn.Transaction(func(tx *storage.Connection) error {
		delModels := map[string]interface{}{
			"user":          &User{},
			"refresh token": &RefreshToken{},
		}
		for name, dm := range delModels {
			if err := tx.Delete(dm, " instance_id = ?", instance.ID).Error; err != nil {
				return errors.Wrapf(err, "Error deleting %s records", name)
			}
		}

		return errors.Wrap(tx.Delete(instance).Error, "Error deleting instance record")
	})
}

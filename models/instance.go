package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/netlify/gotrue/conf"
)

const baseConfigKey = ""

type Instance struct {
	ID string `json:"id"`
	// Netlify UUID
	UUID string `json:"uuid,omitempty"`

	// force usage of text column type
	RawBaseConfig string              `json:"-" gorm:"size:65535"`
	BaseConfig    *conf.Configuration `json:"config"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// TableName returns the table name used for the Instance model
func (i *Instance) TableName() string {
	return tableName("instances")
}

// AfterFind database callback.
func (i *Instance) AfterFind() error {
	if i.RawBaseConfig != "" {
		err := json.Unmarshal([]byte(i.RawBaseConfig), &i.BaseConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

// BeforeSave database callback.
func (i *Instance) BeforeSave() error {
	if i.BaseConfig != nil {
		data, err := json.Marshal(i.BaseConfig)
		if err != nil {
			return err
		}
		i.RawBaseConfig = string(data)
	}
	return nil
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

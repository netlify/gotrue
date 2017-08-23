package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/imdario/mergo"
	"github.com/netlify/gotrue/conf"
)

const baseConfigKey = ""

type Instance struct {
	ID string `json:"id" bson:"_id,omitempty"`
	// Netlify UUID
	UUID string `json:"uuid,omitempty" bson:"uuid,omitempty"`

	// force usage of text column type
	RawBaseConfig string              `json:"-" bson:"-" gorm:"size:65535"`
	BaseConfig    *conf.Configuration `json:"config"`

	RawContexts string                        `json:"-" bson:"-"`
	Contexts    map[string]conf.Configuration `sql:"-" json:"contexts" bson:"contexts"`

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
	if i.RawContexts != "" {
		err := json.Unmarshal([]byte(i.RawContexts), &i.Contexts)
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
	if i.Contexts != nil {
		data, err := json.Marshal(i.Contexts)
		if err != nil {
			return err
		}
		i.RawContexts = string(data)
	}
	return nil
}

// ConfigForEnvironment loads the configuration for an environment, merging in
// the base configuration values.
func (i *Instance) ConfigForEnvironment(env string) (*conf.Configuration, error) {
	if i.BaseConfig == nil {
		return nil, errors.New("no configuration data available")
	}

	baseConf := conf.Configuration{}
	if err := mergo.MergeWithOverwrite(&baseConf, i.BaseConfig); err != nil {
		return nil, err
	}
	baseConf.ApplyDefaults()

	if i.Contexts != nil {
		envConf, ok := i.Contexts[env]
		if !ok {
			return &baseConf, nil
		}
		if err := mergo.MergeWithOverwrite(&baseConf, envConf); err != nil {
			return nil, err
		}
	}

	return &baseConf, nil
}

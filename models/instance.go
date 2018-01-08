package models

import (
	"errors"
	"time"

	"github.com/netlify/gotrue/conf"
	uuid "github.com/satori/go.uuid"
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

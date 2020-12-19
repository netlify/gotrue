package storage

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
)

const (
	globalConfigCtxKey = "global_config"
	instanceIDCtxKey   = "instance_id"
)

func (c *Connection) withContext(ctx context.Context, global *conf.GlobalConfiguration, instanceID uuid.UUID) *Connection {
	ctx = withGlobalConfig(ctx, global)
	ctx = withInstanceID(ctx, instanceID)
	return &Connection{DB: c.DB.WithContext(ctx)}
}

// withConfig adds the tenant configuration to the context.
func withGlobalConfig(ctx context.Context, config *conf.GlobalConfiguration) context.Context {
	return context.WithValue(ctx, globalConfigCtxKey, config)
}

func (c *Connection) GetGlobalConfig(ctx context.Context) *conf.GlobalConfiguration {
	obj, found := c.Get(globalConfigCtxKey)
	if !found {
		return nil
	}
	return obj.(*conf.GlobalConfiguration)
}

// withInstanceID adds the instance id to the context.
func withInstanceID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, instanceIDCtxKey, id)
}

// GetInstanceID reads the instance id from the context.
func (c *Connection) GetInstanceID(ctx context.Context) uuid.UUID {
	obj, found := c.Get(instanceIDCtxKey)
	if !found {
		return uuid.Nil
	}
	return obj.(uuid.UUID)
}

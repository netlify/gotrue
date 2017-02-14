package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type RefreshToken struct {
	ID  int64         `bson:"seq_id,omitempty"`
	BID bson.ObjectId `bson:"_id" sql:"-"`

	Token string `bson:"token"`

	User   User   `bson:"-"`
	UserID string `bson:"user_id"`

	Revoked   bool      `bson:"revoked"`
	CreatedAt time.Time `bson:"created_at"`
}

func (RefreshToken) TableName() string {
	return tableName("refresh_tokens")
}

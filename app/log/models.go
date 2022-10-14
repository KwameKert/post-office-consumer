package log

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Log struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	Data      string             `bson:"data" json:"data"`
	Domain    primitive.ObjectID `bson:"domain" json:"domain_id"`
	Action    string             `bson:"action"`
	Creator   string             `bson:"user_id"`
}

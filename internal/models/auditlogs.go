package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type AuditLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Entity      string             `bson:"entity" json:"entity"`
	Action      string             `bson:"action" json:"action"`
	PerformedBy string             `bson:"performed_by" json:"performed_by"` // could be user ID or system
	Data        any                `bson:"data" json:"data"`                 // raw payload
}

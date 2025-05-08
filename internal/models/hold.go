package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Hold struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MemberID    primitive.ObjectID `bson:"member_id" json:"member_id"`
	CopyBarcode string             `bson:"copy_barcode" json:"copy_barcode"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Fulfilled   bool               `bson:"fulfilled" json:"fulfilled"`
	Notified    bool               `bson:"notified" json:"notified"`
	PickupBy    *time.Time         `bson:"pickup_by,omitempty" json:"pickup_by,omitempty"`
}

const (
	HoldEntity = "hold"
)

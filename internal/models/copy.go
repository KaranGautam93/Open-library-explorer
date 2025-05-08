package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CopyStatus string

const (
	StatusAvailable CopyStatus = "AVAILABLE"
	StatusOnLoan    CopyStatus = "ON_LOAN"
	StatusReserved  CopyStatus = "RESERVED"
	StatusLost      CopyStatus = "LOST"

	CopyEntity = "Copy"
)

type Copy struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ISBN      string             `bson:"isbn" json:"isbn"`
	Barcode   string             `bson:"barcode" json:"barcode"`
	Status    CopyStatus         `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

var ValidCopyStatuses = map[string]bool{
	string(StatusAvailable): true,
	string(StatusOnLoan):    true,
	string(StatusReserved):  true,
	string(StatusLost):      true,
}

func IsValidCopyStatus(status string) bool {
	return ValidCopyStatuses[status]
}

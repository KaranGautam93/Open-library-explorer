package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Loan struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MemberID    primitive.ObjectID `bson:"member_id" json:"member_id"`
	CopyBarcode string             `bson:"copy_barcode" json:"copy_barcode"`
	LoanDate    time.Time          `bson:"loan_date" json:"loan_date"`
	DueDate     time.Time          `bson:"due_date" json:"due_date"`
	Returned    bool               `bson:"returned" json:"returned"`
}

const (
	LoanEntity = "loan"
)

package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type MembershipTier string

const (
	TierStandard MembershipTier = "STANDARD"
	TierPremium  MembershipTier = "PREMIUM"

	MemberEntity = "member"
)

type Member struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Phone     string             `bson:"phone" json:"phone"`
	Tier      MembershipTier     `bson:"tier" json:"tier"`
	Blocked   bool               `bson:"blocked" json:"blocked"`
	Active    bool               `bson:"active" json:"active"` // For deactivation
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

var MemberTierMap = map[string]bool{
	string(TierStandard): true,
	string(TierPremium):  true,
}

func IsValidMemberTier(tier string) bool {
	return MemberTierMap[tier]
}

var TierRenewalDays = map[string]int{
	string(TierStandard): 7,
	string(TierPremium):  14,
}

package handlers

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"open-library-explorer/internal/constants"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"
)

type LoanHandler struct {
	MemberCol      *mongo.Collection
	CopyCol        *mongo.Collection
	LoanCol        *mongo.Collection
	ReservationCol *mongo.Collection
	AuditLogger    utils.Logger
	Config         struct {
		PremiumMemberRenewalDays  int
		StandardMemberRenewalDays int
	}
}

type CheckOutRequest struct {
	MemberID    string `json:"member_id"`
	CopyBarcode string `json:"copy_barcode"`
}

func (h *LoanHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	var req CheckOutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	memberID, err := primitive.ObjectIDFromHex(req.MemberID)
	if err != nil {
		utils.JSONError(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Fetch member
	var member models.Member
	if err := h.MemberCol.FindOne(r.Context(), bson.M{"_id": memberID}).Decode(&member); err != nil {
		utils.JSONError(w, "Member not found", http.StatusNotFound)
		return
	}
	if member.Blocked {
		utils.JSONError(w, "Member is blocked", http.StatusForbidden)
		return
	}

	// Fetch copyObj
	var copyObj models.Copy
	if err := h.CopyCol.FindOne(r.Context(), bson.M{"barcode": req.CopyBarcode}).Decode(&copyObj); err != nil {
		utils.JSONError(w, "Copy not found", http.StatusNotFound)
		return
	}
	if copyObj.Status != models.StatusAvailable {
		utils.JSONError(w, "Copy not available", http.StatusConflict)
		return
	}

	// Determine due date
	var loanDays int
	switch member.Tier {
	case models.TierPremium:
		loanDays = h.Config.PremiumMemberRenewalDays
	case models.TierStandard:
		loanDays = h.Config.StandardMemberRenewalDays
	default:
		loanDays = 7
	}
	now := time.Now()
	loan := models.Loan{
		ID:          primitive.NewObjectID(),
		MemberID:    memberID,
		CopyBarcode: req.CopyBarcode,
		LoanDate:    now,
		DueDate:     now.AddDate(0, 0, loanDays),
		Returned:    false,
	}

	// Insert loan
	_, err = h.LoanCol.InsertOne(r.Context(), loan)
	if err != nil {
		utils.JSONError(w, "Failed to record loan", http.StatusInternalServerError)
		return
	}

	// Update copyObj status
	_, err = h.CopyCol.UpdateOne(r.Context(),
		bson.M{"barcode": req.CopyBarcode},
		bson.M{"$set": bson.M{"status": models.StatusOnLoan}},
	)
	if err != nil {
		utils.JSONError(w, "Failed to update copyObj status", http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(context.Background(), models.LoanEntity, constants.CheckOut, loan)
	json.NewEncoder(w).Encode(loan)
}

func (h *LoanHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CopyBarcode string `json:"copy_barcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// 1. Find active loan
	filter := bson.M{
		"copy_barcode": req.CopyBarcode,
		"returned":     false,
	}
	update := bson.M{
		"$set": bson.M{"returned": true},
	}
	result := h.LoanCol.FindOneAndUpdate(r.Context(), filter, update)
	if result.Err() != nil {
		utils.JSONError(w, "Active loan not found for this copy", http.StatusNotFound)
		return
	}

	// 2. Check for existing reservation
	var hold models.Hold
	err := h.ReservationCol.FindOne(r.Context(), bson.M{
		"copy_barcode": req.CopyBarcode,
		"fulfilled":    false,
		"notified":     false,
	}, options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: 1}})).Decode(&hold)

	newStatus := models.StatusAvailable
	if err == nil {
		newStatus = models.StatusReserved

		// Mark as notified (mock)
		_, _ = h.ReservationCol.UpdateOne(r.Context(),
			bson.M{"_id": hold.ID},
			bson.M{"$set": bson.M{"notified": true}},
		)

		utils.AppendToEmailLog(r.Context(), hold.MemberID.Hex(), req.CopyBarcode)
	}

	// 3. Update copy status
	_, err = h.CopyCol.UpdateOne(r.Context(),
		bson.M{"barcode": req.CopyBarcode},
		bson.M{"$set": bson.M{"status": newStatus}},
	)
	if err != nil {
		utils.JSONError(w, "Failed to update copy status", http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(context.Background(), models.LoanEntity, constants.CheckIn, req.CopyBarcode)

	json.NewEncoder(w).Encode(bson.M{
		"message": "Check-in successful",
		"status":  newStatus,
	})
}

func (h *LoanHandler) RenewLoan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemberID    string `json:"member_id"`
		CopyBarcode string `json:"copy_barcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	memberOID, err := primitive.ObjectIDFromHex(req.MemberID)
	if err != nil {
		utils.JSONError(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// 1. Load member
	var member models.Member
	if err := h.MemberCol.FindOne(r.Context(), bson.M{"_id": memberOID}).Decode(&member); err != nil {
		utils.JSONError(w, "Member not found", http.StatusNotFound)
		return
	}

	// 2. Find active loan
	var loan models.Loan
	err = h.LoanCol.FindOne(r.Context(), bson.M{
		"copy_barcode": req.CopyBarcode,
		"member_id":    memberOID,
		"returned":     false,
	}).Decode(&loan)
	if err != nil {
		utils.JSONError(w, "Active loan not found for this member and copy", http.StatusNotFound)
		return
	}

	// 3. Check if hold exists
	count, err := h.ReservationCol.CountDocuments(r.Context(), bson.M{
		"copy_barcode": req.CopyBarcode,
		"fulfilled":    false,
	})
	if err != nil {
		utils.JSONError(w, "Error checking holds", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		utils.JSONError(w, "Renewal not allowed — reservations exist", http.StatusForbidden)
		return
	}

	// 4. Compute new due date based on tier
	days, ok := models.TierRenewalDays[string(member.Tier)]

	switch member.Tier {
	case models.TierPremium:
		days = h.Config.PremiumMemberRenewalDays
	case models.TierStandard:
		days = h.Config.StandardMemberRenewalDays
	default:
		days = 7
	}
	if !ok {
		utils.JSONError(w, "Unknown membership tier", http.StatusInternalServerError)
		return
	}
	newDue := loan.DueDate.AddDate(0, 0, days)

	// 5. Update due_date
	_, err = h.LoanCol.UpdateOne(r.Context(),
		bson.M{"_id": loan.ID},
		bson.M{"$set": bson.M{"due_date": newDue}},
	)
	if err != nil {
		utils.JSONError(w, "Failed to renew loan", http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(context.Background(), models.LoanEntity, constants.RenewLoan, loan)

	json.NewEncoder(w).Encode(bson.M{
		"message":   "Loan renewed",
		"new_due":   newDue.Format(time.RFC3339),
		"member_id": req.MemberID,
	})
}

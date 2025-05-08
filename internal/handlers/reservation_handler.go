package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"open-library-explorer/internal/constants"
	"time"

	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ReservationHandler struct {
	ReservationCol *mongo.Collection
	CopyCol        *mongo.Collection
	MemberCol      *mongo.Collection
	AuditLogger    utils.Logger
}

func (h *ReservationHandler) PlaceHold(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemberID    string `json:"member_id"`
		CopyBarcode string `json:"copy_barcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	memberID, err := primitive.ObjectIDFromHex(req.MemberID)
	if err != nil {
		utils.JSONError(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// 1. Validate member exists
	var member models.Member
	err = h.MemberCol.FindOne(r.Context(), bson.M{"_id": memberID}).Decode(&member)
	if err != nil {
		utils.JSONError(w, "Member not found", http.StatusNotFound)
		return
	}

	// 1. Check copy is not AVAILABLE
	var copy models.Copy
	err = h.CopyCol.FindOne(r.Context(), bson.M{"barcode": req.CopyBarcode}).Decode(&copy)
	if err != nil {
		utils.JSONError(w, "Copy not found", http.StatusNotFound)
		return
	}

	if copy.Status == models.StatusAvailable {
		utils.JSONError(w, "Copy is available — no need to hold", http.StatusBadRequest)
		return
	}

	// 2. Check if hold already exists for this member+copy
	count, err := h.ReservationCol.CountDocuments(r.Context(), bson.M{
		"member_id":    memberID,
		"copy_barcode": req.CopyBarcode,
		"fulfilled":    false,
	})
	if err != nil {
		utils.JSONError(w, "Error checking existing holds", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		utils.JSONError(w, "Hold already exists for this copy", http.StatusConflict)
		return
	}

	// 3. Insert new hold
	hold := models.Hold{
		MemberID:    memberID,
		CopyBarcode: req.CopyBarcode,
		Timestamp:   time.Now(),
		Fulfilled:   false,
		Notified:    false,
	}

	_, err = h.ReservationCol.InsertOne(r.Context(), hold)
	if err != nil {
		utils.JSONError(w, "Failed to place hold", http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(context.Background(), models.HoldEntity, constants.Create, hold)

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hold placed successfully",
	})
}

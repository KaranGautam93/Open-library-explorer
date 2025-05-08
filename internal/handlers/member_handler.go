package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"open-library-explorer/internal/constants"
	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"
	"time"
)

type MemberHandler struct {
	Collection  *mongo.Collection
	AuditLogger utils.Logger
}

func NewMemberHandler(coll *mongo.Collection, logger utils.Logger) *MemberHandler {
	return &MemberHandler{Collection: coll, AuditLogger: logger}
}

func (h *MemberHandler) RegisterMember(w http.ResponseWriter, r *http.Request) {
	var member models.Member
	if err := json.NewDecoder(r.Body).Decode(&member); err != nil {
		utils.JSONError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	fmt.Println(member.Tier, models.IsValidMemberTier(string(member.Tier)))

	if !models.IsValidMemberTier(string(member.Tier)) {
		utils.JSONError(w, "Invalid member tier", http.StatusBadRequest)
		return
	}

	member.ID = primitive.NewObjectID()
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.Collection.InsertOne(ctx, member)
	if err != nil {
		utils.JSONError(w, "Insert failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(ctx, models.MemberEntity, constants.Create, member)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(member)
}

func (h *MemberHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	memberID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.JSONError(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	var updateData bson.M
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		utils.JSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if tier, ok := updateData["tier"]; ok && !models.IsValidMemberTier(tier.(string)) {
		utils.JSONError(w, "Invalid tier", http.StatusBadRequest)
		return
	}

	updateData["updated_at"] = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := h.Collection.UpdateByID(ctx, memberID, bson.M{"$set": updateData})
	if err != nil {
		utils.JSONError(w, "Update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if res.MatchedCount == 0 {
		utils.JSONError(w, "Member not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.AuditLogger.Log(ctx, models.MemberEntity, constants.Update, updateData)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member updated"})
}

func (h *MemberHandler) DeactivateMember(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	memberID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.JSONError(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := h.Collection.UpdateByID(ctx, memberID, bson.M{"$set": bson.M{
		"blocked":    true,
		"updated_at": time.Now(),
	}})
	if err != nil {
		utils.JSONError(w, "Deactivate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if res.MatchedCount == 0 {
		utils.JSONError(w, "Member not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.AuditLogger.Log(ctx, models.MemberEntity, constants.Deactivate, idStr)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member deactivated"})
}

package handlers

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"open-library-explorer/internal/constants"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"
)

type CopyHandler struct {
	Collection  *mongo.Collection
	AuditLogger utils.Logger
}

// POST /copies
func (h *CopyHandler) AddCopy(w http.ResponseWriter, r *http.Request) {
	var copyObj models.Copy
	if err := json.NewDecoder(r.Body).Decode(&copyObj); err != nil {
		utils.JSONError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	copyObj.CreatedAt = time.Now()
	copyObj.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := h.Collection.InsertOne(ctx, copyObj)
	if err != nil {
		utils.JSONError(w, "Insert failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	copyObj.ID = res.InsertedID.(primitive.ObjectID)

	h.AuditLogger.Log(ctx, models.CopyEntity, constants.Create, copyObj)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(copyObj)
}

// GET /copies?isbn=xxx
func (h *CopyHandler) GetCopies(w http.ResponseWriter, r *http.Request) {
	isbn := r.URL.Query().Get("isbn")
	filter := bson.M{}
	if isbn != "" {
		filter["isbn"] = isbn
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := h.Collection.Find(ctx, filter)
	if err != nil {
		utils.JSONError(w, "Failed to fetch copies", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var copies []models.Copy
	if err = cursor.All(ctx, &copies); err != nil {
		utils.JSONError(w, "Error decoding result", http.StatusInternalServerError)
		return
	}

	if len(copies) == 0 {
		utils.JSONError(w, "No copies found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(copies)
}

// PUT /copies/{barcode}
func (h *CopyHandler) UpdateCopy(w http.ResponseWriter, r *http.Request) {
	barcode := mux.Vars(r)["barcode"]

	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		utils.JSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if statusVal, ok := updateData["status"]; ok {
		statusStr, ok := statusVal.(string)
		if !ok || !models.IsValidCopyStatus(statusStr) {
			utils.JSONError(w, "Invalid status value", http.StatusBadRequest)
			return
		}
	}

	updateData["updated_at"] = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.Collection.UpdateOne(
		ctx,
		bson.M{"barcode": barcode},
		bson.M{"$set": updateData},
	)

	if err != nil {
		utils.JSONError(w, "Update failed", http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		utils.JSONError(w, "Copy not found", http.StatusNotFound)
		return
	}

	h.AuditLogger.Log(ctx, models.CopyEntity, constants.Create, updateData)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Copy updated",
	})
}

// DELETE /copies/{barcode}
func (h *CopyHandler) DeleteCopy(w http.ResponseWriter, r *http.Request) {
	barcode := mux.Vars(r)["barcode"]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.Collection.DeleteOne(ctx, bson.M{"barcode": barcode})
	if err != nil {
		utils.JSONError(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	if result.DeletedCount == 0 {
		utils.JSONError(w, "Copy not found", http.StatusNotFound)
		return
	}

	h.AuditLogger.Log(ctx, models.CopyEntity, constants.Create, barcode)

	w.WriteHeader(http.StatusNoContent)
}

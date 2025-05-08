package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"open-library-explorer/internal/constants"
	"open-library-explorer/internal/utils"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"open-library-explorer/internal/models"
)

type BookHandler struct {
	BookCollection *mongo.Collection
	CopyCollection *mongo.Collection
	AuditLogger    utils.Logger
}

func NewBookHandler(bookColl, copyColl *mongo.Collection, logger utils.Logger) *BookHandler {
	return &BookHandler{
		BookCollection: bookColl,
		CopyCollection: copyColl,
		AuditLogger:    logger,
	}
}

// POST /books
func (h *BookHandler) AddBook(w http.ResponseWriter, r *http.Request) {
	var book models.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		utils.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.BookCollection.InsertOne(ctx, book)
	if err != nil {
		utils.JSONError(w, "Insert failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.AuditLogger.Log(ctx, models.BookEntity, constants.Create, book)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(book)
}

// GET /books
func (h *BookHandler) GetBooks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := h.BookCollection.Find(ctx, bson.M{})
	if err != nil {
		utils.JSONError(w, "Failed to fetch books", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var books []models.Book
	if err = cursor.All(ctx, &books); err != nil {
		utils.JSONError(w, "Error decoding books", http.StatusInternalServerError)
		return
	}

	if len(books) == 0 {
		utils.JSONError(w, "No books found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(books)
}

// GET /books/{isbn}
func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	isbn := mux.Vars(r)["isbn"]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var book models.Book
	err := h.BookCollection.FindOne(ctx, bson.M{"isbn": isbn}).Decode(&book)
	if err != nil {
		utils.JSONError(w, "Book not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(book)
}

// PUT /books/{isbn}
func (h *BookHandler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	isbn := mux.Vars(r)["isbn"]

	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		utils.JSONError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if len(updateData) == 0 {
		utils.JSONError(w, "No update fields provided", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.BookCollection.UpdateOne(
		ctx,
		bson.M{"isbn": isbn},
		bson.M{"$set": updateData},
	)

	if err != nil {
		utils.JSONError(w, "Update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		utils.JSONError(w, "Book not found", http.StatusNotFound)
		return
	}

	h.AuditLogger.Log(ctx, models.BookEntity, constants.Update, updateData)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Book updated successfully",
		"modifiedCount": result.ModifiedCount,
	})
}

// DELETE /books/{isbn}
func (h *BookHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	isbn := mux.Vars(r)["isbn"]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.BookCollection.DeleteOne(ctx, bson.M{"isbn": isbn})
	if err != nil {
		utils.JSONError(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	if result.DeletedCount == 0 {
		utils.JSONError(w, "Book not found", http.StatusNotFound)
		return
	}

	h.AuditLogger.Log(ctx, models.BookEntity, constants.Delete, isbn)

	w.WriteHeader(http.StatusNoContent)
}

func (h *BookHandler) SearchBooks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	statusFilter := r.URL.Query().Get("status")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}

	if query != "" {
		filter["$text"] = bson.M{"$search": query}
	}

	if statusFilter != "" {
		if !models.IsValidCopyStatus(statusFilter) {
			utils.JSONError(w, "Invalid status", http.StatusInternalServerError)
			return
		}
		copiesColl := h.CopyCollection

		isbnList, err := copiesColl.Distinct(ctx, "isbn", bson.M{"status": statusFilter})
		if err != nil {
			utils.JSONError(w, "Failed to query copies: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if len(isbnList) == 0 {
			utils.JSONError(w, "No record found", http.StatusNotFound)
			return
		}
		filter["isbn"] = bson.M{"$in": isbnList}
	}

	cursor, err := h.BookCollection.Find(ctx, filter)
	if err != nil {
		utils.JSONError(w, "Failed to search books: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var results []models.Book
	if err = cursor.All(ctx, &results); err != nil {
		utils.JSONError(w, "Failed to decode books", http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		utils.JSONError(w, "No record found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(results)
}

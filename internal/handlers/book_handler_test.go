package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"open-library-explorer/internal/handlers"
	"open-library-explorer/internal/models"
)

func TestBookHandler_AddBook(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful book addition", func(mt *mtest.T) {
		handler := handlers.BookHandler{
			BookCollection: mt.Coll,
		}

		router := mux.NewRouter()
		router.HandleFunc("/books", handler.AddBook).Methods("POST")

		newBook := models.Book{
			Title: "Test Book",
			ISBN:  "978-3-16-148410-0",
		}

		reqBytes, _ := json.Marshal(newBook)
		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status Created, got %v", res.Status)
		}
	})

	mt.Run("invalid book data", func(mt *mtest.T) {
		handler := handlers.BookHandler{
			BookCollection: mt.Coll,
		}

		router := mux.NewRouter()
		router.HandleFunc("/books", handler.AddBook).Methods("POST")

		// Sending an empty JSON body
		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status BadRequest, got %v", res.Status)
		}
	})
}

func TestBookHandler_GetBooks(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful books retrieval", func(mt *mtest.T) {
		handler := handlers.BookHandler{
			BookCollection: mt.Coll,
		}

		router := mux.NewRouter()
		router.HandleFunc("/books", handler.GetBooks).Methods("GET")

		// Mock response for books
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.books", mtest.FirstBatch, bson.D{
			{Key: "title", Value: "Test Book"},
			{Key: "isbn", Value: "978-3-16-148410-0"},
		}))

		req := httptest.NewRequest(http.MethodGet, "/books", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})
}

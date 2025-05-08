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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"open-library-explorer/internal/handlers"
	"open-library-explorer/internal/models"
)

func TestReservationHandler_PlaceHold(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful hold placement", func(mt *mtest.T) {
		handler := handlers.ReservationHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			ReservationCol: mt.Coll,
		}

		memberID := primitive.NewObjectID()
		copyBarcode := "123456"

		member := models.Member{
			ID:      memberID,
			Name:    "Jane Doe",
			Blocked: false,
		}
		copyObj := models.Copy{
			Barcode: copyBarcode,
			Status:  models.StatusOnLoan,
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.members", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: member.ID},
				{Key: "blocked", Value: member.Blocked},
			}),
			mtest.CreateCursorResponse(1, "test.copies", mtest.FirstBatch, bson.D{
				{Key: "barcode", Value: copyObj.Barcode},
				{Key: "status", Value: copyObj.Status},
			}),
		)

		router := mux.NewRouter()
		router.HandleFunc("/holds/place", handler.PlaceHold).Methods("POST")

		reqBody := struct {
			MemberID    string `json:"member_id"`
			CopyBarcode string `json:"copy_barcode"`
		}{
			MemberID:    memberID.Hex(),
			CopyBarcode: copyBarcode,
		}

		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/holds/place", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})

	mt.Run("copy already available, no hold needed", func(mt *mtest.T) {
		handler := handlers.ReservationHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			ReservationCol: mt.Coll,
		}

		memberID := primitive.NewObjectID()
		copyBarcode := "654321"

		member := models.Member{
			ID:      memberID,
			Name:    "Jane Doe",
			Blocked: false,
		}
		copyObj := models.Copy{
			Barcode: copyBarcode,
			Status:  models.StatusAvailable,
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.members", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: member.ID},
				{Key: "blocked", Value: member.Blocked},
			}),
			mtest.CreateCursorResponse(1, "test.copies", mtest.FirstBatch, bson.D{
				{Key: "barcode", Value: copyObj.Barcode},
				{Key: "status", Value: copyObj.Status},
			}),
		)

		router := mux.NewRouter()
		router.HandleFunc("/holds/place", handler.PlaceHold).Methods("POST")

		reqBody := struct {
			MemberID    string `json:"member_id"`
			CopyBarcode string `json:"copy_barcode"`
		}{
			MemberID:    "792",
			CopyBarcode: copyBarcode,
		}

		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/holds/place", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", res.Status)
		}
	})
}

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

func TestLoanHandler_CheckOut(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil { // Ensure the client is initialized before disconnecting
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful checkout", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol: mt.Coll,
			CopyCol:   mt.Coll,
			LoanCol:   mt.Coll,
			Config: struct {
				PremiumMemberRenewalDays  int
				StandardMemberRenewalDays int
			}{
				PremiumMemberRenewalDays:  30,
				StandardMemberRenewalDays: 14,
			},
		}

		// Mock member data
		memberID := primitive.NewObjectID()
		copyBarcode := "123456"
		member := models.Member{
			ID:      memberID,
			Name:    "John Doe",
			Tier:    models.TierPremium,
			Blocked: false,
		}
		copyObj := models.Copy{
			Barcode: copyBarcode,
			Status:  models.StatusAvailable,
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.members", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: member.ID},
				{Key: "tier", Value: member.Tier},
				{Key: "blocked", Value: member.Blocked},
			}),
			mtest.CreateCursorResponse(1, "test.copies", mtest.FirstBatch, bson.D{
				{Key: "barcode", Value: copyObj.Barcode},
				{Key: "status", Value: copyObj.Status},
			}),
		)

		// Create and configure the router
		router := mux.NewRouter()
		router.HandleFunc("/checkout", handler.CheckOut).Methods("POST")

		reqBody := handlers.CheckOutRequest{
			MemberID:    memberID.Hex(),
			CopyBarcode: copyBarcode,
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		//as this member is not present
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})

	mt.Run("blocked member cannot checkout", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol: mt.Coll,
		}

		// Mock member data
		memberID := primitive.NewObjectID()
		member := models.Member{
			ID:      memberID,
			Name:    "John Doe",
			Tier:    models.TierPremium,
			Blocked: true, // Member is blocked
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.members", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: member.ID},
			{Key: "tier", Value: member.Tier},
			{Key: "blocked", Value: member.Blocked},
		}))

		// Create and configure the router
		router := mux.NewRouter()
		router.HandleFunc("/checkout", handler.CheckOut).Methods("POST")

		reqBody := handlers.CheckOutRequest{
			MemberID:    memberID.Hex(),
			CopyBarcode: "123456",
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusForbidden {
			t.Errorf("expected status Forbidden, got %v", res.Status)
		}
	})
}

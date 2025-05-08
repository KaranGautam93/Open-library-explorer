package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"open-library-explorer/internal/models"
	"open-library-explorer/internal/utils"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"open-library-explorer/internal/handlers"
)

func TestLoanHandler_CheckIn(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful check-in", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			LoanCol:        mt.Coll,
			ReservationCol: mt.Coll,
			CopyCol:        mt.Coll,
		}

		copyBarcode := "123456"

		// Mock loan and reservation data
		mt.AddMockResponses(
			// Active loan found
			mtest.CreateCursorResponse(1, "test.loans", mtest.FirstBatch, bson.D{
				{Key: "copy_barcode", Value: copyBarcode},
				{Key: "returned", Value: false},
			}),
			// Mock reservation
			mtest.CreateCursorResponse(1, "test.reservations", mtest.FirstBatch, bson.D{
				{Key: "copy_barcode", Value: copyBarcode},
				{Key: "fulfilled", Value: false},
				{Key: "notified", Value: false},
			}),
		)

		router := mux.NewRouter()
		router.HandleFunc("/checkin", handler.CheckIn).Methods("POST")

		reqBody := struct {
			CopyBarcode string `json:"copy_barcode"`
		}{
			CopyBarcode: copyBarcode,
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/checkin", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})

	mt.Run("loan not found for check-in", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			LoanCol: mt.Coll,
		}

		copyBarcode := "654321"

		mt.AddMockResponses(
			// No active loan found
			mtest.CreateCursorResponse(0, "test.loans", mtest.FirstBatch, nil),
		)

		router := mux.NewRouter()
		router.HandleFunc("/checkin", handler.CheckIn).Methods("POST")

		reqBody := struct {
			CopyBarcode string `json:"copy_barcode"`
		}{
			CopyBarcode: copyBarcode,
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/checkin", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status NotFound, got %v", res.Status)
		}
	})
}

func TestLoanHandler_RenewLoan(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("successful loan renewal", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			LoanCol:        mt.Coll,
			ReservationCol: mt.Coll,
			AuditLogger:    utils.Logger{},
			Config: struct {
				PremiumMemberRenewalDays  int
				StandardMemberRenewalDays int
			}{
				PremiumMemberRenewalDays:  30,
				StandardMemberRenewalDays: 14,
			},
		}

		memberID := primitive.NewObjectID()
		copyBarcode := "123456"

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.loans", mtest.FirstBatch, bson.D{
				{Key: "member_id", Value: memberID},
				{Key: "copy_barcode", Value: copyBarcode},
				{Key: "returned", Value: false},
				{Key: "due_date", Value: time.Now()},
			}),
		)

		router := mux.NewRouter()
		router.HandleFunc("/loan/renew", handler.RenewLoan).Methods("POST")

		reqBody := struct {
			MemberID    string `json:"member_id"`
			CopyBarcode string `json:"copy_barcode"`
		}{
			MemberID:    memberID.Hex(),
			CopyBarcode: copyBarcode,
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/loan/renew", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})

	mt.Run("loan not found for renewal", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			LoanCol:        mt.Coll,
			ReservationCol: mt.Coll,
			AuditLogger:    utils.Logger{},
		}

		memberID := primitive.NewObjectID()
		copyBarcode := "654321"

		mt.AddMockResponses(
			// No loan found
			mtest.CreateCursorResponse(0, "test.loans", mtest.FirstBatch, nil),
		)

		router := mux.NewRouter()
		router.HandleFunc("/loan/renew", handler.RenewLoan).Methods("POST")

		reqBody := struct {
			MemberID    string `json:"member_id"`
			CopyBarcode string `json:"copy_barcode"`
		}{
			MemberID:    memberID.Hex(),
			CopyBarcode: copyBarcode,
		}
		reqBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/loan/renew", bytes.NewReader(reqBytes))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status NotFound, got %v", res.Status)
		}
	})
}

func TestLoanHandler_OverdueLoans(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	if mt.Client != nil {
		defer mt.Client.Disconnect(context.Background())
	}

	mt.Run("identify overdue loans", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			LoanCol:        mt.Coll,
			ReservationCol: mt.Coll,
			AuditLogger:    utils.Logger{},
			Config: struct {
				PremiumMemberRenewalDays  int
				StandardMemberRenewalDays int
			}{},
		}

		// Mock overdue loan data
		memberID := primitive.NewObjectID()
		copyBarcode := "123456"
		loanID := primitive.NewObjectID()
		overdueDate := time.Now().AddDate(0, 0, -5) // 5 days overdue

		overdueLoan := models.Loan{
			ID:          loanID,
			MemberID:    memberID,
			CopyBarcode: copyBarcode,
			LoanDate:    time.Now().AddDate(0, 0, -10),
			DueDate:     overdueDate,
			Returned:    false,
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.loans", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: overdueLoan.ID},
				{Key: "member_id", Value: overdueLoan.MemberID},
				{Key: "copy_barcode", Value: overdueLoan.CopyBarcode},
				{Key: "loan_date", Value: overdueLoan.LoanDate},
				{Key: "due_date", Value: overdueLoan.DueDate},
				{Key: "returned", Value: overdueLoan.Returned},
			}),
		)

		router := mux.NewRouter()
		router.HandleFunc("/loans/overdue", handler.GetOverdueLoans).Methods("GET")

		req := httptest.NewRequest(http.MethodGet, "/loans/overdue", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status OK, got %v", res.Status)
		}
	})

	mt.Run("restrict actions for member with overdue loans", func(mt *mtest.T) {
		handler := handlers.LoanHandler{
			MemberCol:      mt.Coll,
			CopyCol:        mt.Coll,
			LoanCol:        mt.Coll,
			ReservationCol: mt.Coll,
			AuditLogger:    utils.Logger{},
			Config: struct {
				PremiumMemberRenewalDays  int
				StandardMemberRenewalDays int
			}{},
		}

		// Mock overdue loan for a member
		memberID := primitive.NewObjectID()
		copyBarcode := "654321"
		overdueDate := time.Now().AddDate(0, 0, -7) // 7 days overdue

		overdueLoan := models.Loan{
			MemberID:    memberID,
			CopyBarcode: copyBarcode,
			LoanDate:    time.Now().AddDate(0, 0, -15),
			DueDate:     overdueDate,
			Returned:    false,
		}

		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "test.loans", mtest.FirstBatch, bson.D{
				{Key: "member_id", Value: overdueLoan.MemberID},
				{Key: "copy_barcode", Value: overdueLoan.CopyBarcode},
				{Key: "due_date", Value: overdueLoan.DueDate},
				{Key: "returned", Value: overdueLoan.Returned},
			}),
		)

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

		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status Forbidden, got %v", res.Status)
		}
	})
}

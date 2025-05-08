package handlers

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"open-library-explorer/internal/models"
	"time"
)

type MetricsHandler struct {
	CopyCol   *mongo.Collection
	MemberCol *mongo.Collection
	LoanCol   *mongo.Collection
	Config    struct {
		FineRate float64
	}
}

func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	todayStart := time.Now().Truncate(24 * time.Hour)

	// 1. Total books (copies)
	totalBooks, _ := h.CopyCol.CountDocuments(ctx, bson.M{})

	// 2. Active members
	activeMembers, _ := h.MemberCol.CountDocuments(ctx, bson.M{
		"blocked": false,
	})

	// 3. Loans today
	loansToday, _ := h.LoanCol.CountDocuments(ctx, bson.M{
		"loan_date": bson.M{
			"$gte": todayStart,
		},
	})

	// 4. Overdue count
	now := time.Now()
	overdueCount, _ := h.LoanCol.CountDocuments(ctx, bson.M{
		"due_date": bson.M{"$lt": now},
		"returned": false,
	})

	// Here we'll assume fine = $1 per overdue per day
	cursor, _ := h.LoanCol.Find(ctx, bson.M{
		"due_date": bson.M{"$lt": now},
		"returned": false,
	})
	var loans []models.Loan
	_ = cursor.All(ctx, &loans)

	finePerDay := h.Config.FineRate
	var fineRevenue float64
	for _, loan := range loans {
		daysLate := int(now.Sub(loan.DueDate).Hours() / 24)
		if daysLate > 0 {
			fineRevenue += float64(daysLate) * finePerDay
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_books":    totalBooks,
		"active_members": activeMembers,
		"loans_today":    loansToday,
		"overdue_count":  overdueCount,
		"fine_revenue":   fineRevenue,
	})
}

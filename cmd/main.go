package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"open-library-explorer/internal/handlers"
	"open-library-explorer/internal/middleware"
	"open-library-explorer/internal/utils"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"

	"open-library-explorer/configs"
	"open-library-explorer/internal/db"
)

func main() {
	cfg := configs.LoadConfig()
	db.Connect(cfg.MongoURI)
	utils.InitJwtSecret(cfg.JWTSecret)

	r := mux.NewRouter()
	r.Use(middleware.JSONMiddleware)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	authHandler := &handlers.AuthHandler{
		ConfigCreds: struct {
			UserId       string
			Username     string
			UserPassword string
		}{UserId: cfg.UserId, Username: cfg.UserName, UserPassword: cfg.UserPassword},
	}
	r.HandleFunc("/login", authHandler.Login).Methods("POST")

	auditCol := db.GetCollection(configs.LoadConfig().DBName, "audit_logs")
	auditLogger := utils.Logger{Collection: auditCol}

	bookColl := db.GetCollection(cfg.DBName, "books")
	copyColl := db.GetCollection(cfg.DBName, "copies")

	bookHandler := handlers.NewBookHandler(bookColl, copyColl, auditLogger)

	booksRouter := r.PathPrefix("/").Subrouter()
	booksRouter.Use(middleware.JWTAuthMiddleware)

	booksRouter.HandleFunc("/books", bookHandler.AddBook).Methods("POST")
	booksRouter.HandleFunc("/books", bookHandler.GetBooks).Methods("GET")
	booksRouter.HandleFunc("/books/search", bookHandler.SearchBooks).Methods("GET")
	booksRouter.HandleFunc("/books/{isbn}", bookHandler.GetBook).Methods("GET")
	booksRouter.HandleFunc("/books/{isbn}", bookHandler.UpdateBook).Methods("PUT")
	booksRouter.HandleFunc("/books/{isbn}", bookHandler.DeleteBook).Methods("DELETE")

	copyColl = db.GetCollection(cfg.DBName, "copies")
	copyHandler := handlers.CopyHandler{Collection: copyColl, AuditLogger: auditLogger}

	r.HandleFunc("/copies", copyHandler.AddCopy).Methods("POST")
	r.HandleFunc("/copies", copyHandler.GetCopies).Methods("GET")
	r.HandleFunc("/copies/{barcode}", copyHandler.UpdateCopy).Methods("PUT")
	r.HandleFunc("/copies/{barcode}", copyHandler.DeleteCopy).Methods("DELETE")

	memberColl := db.GetCollection(cfg.DBName, "members")
	memberHandler := handlers.NewMemberHandler(memberColl, auditLogger)

	r.HandleFunc("/members", memberHandler.RegisterMember).Methods("POST")
	r.HandleFunc("/members/{id}", memberHandler.UpdateMember).Methods("PUT")
	r.HandleFunc("/members/{id}/deactivate", memberHandler.DeactivateMember).Methods("PATCH")

	loanHandler := &handlers.LoanHandler{
		MemberCol:      db.GetCollection(cfg.DBName, "members"),
		CopyCol:        db.GetCollection(cfg.DBName, "copies"),
		LoanCol:        db.GetCollection(cfg.DBName, "loans"),
		ReservationCol: db.GetCollection(cfg.DBName, "holds"),
		AuditLogger:    auditLogger,
		Config: struct {
			PremiumMemberRenewalDays  int
			StandardMemberRenewalDays int
		}{PremiumMemberRenewalDays: cfg.PremiumMembersRenewalDays, StandardMemberRenewalDays: cfg.StandardMembersRenewalDays},
	}

	r.HandleFunc("/checkout", loanHandler.CheckOut).Methods("POST")
	r.HandleFunc("/checkin", loanHandler.CheckIn).Methods("POST")
	r.HandleFunc("/loan/renew", loanHandler.RenewLoan).Methods("POST")
	r.HandleFunc("/loans/overdue", loanHandler.GetOverdueLoans).Methods("GET")

	reservationHandler := &handlers.ReservationHandler{
		ReservationCol: db.GetCollection(cfg.DBName, "holds"),
		CopyCol:        db.GetCollection(cfg.DBName, "copies"),
		MemberCol:      db.GetCollection(cfg.DBName, "members"),
		AuditLogger:    auditLogger,
	}

	r.HandleFunc("/holds/place", reservationHandler.PlaceHold).Methods("POST")

	metricsHandler := handlers.MetricsHandler{
		CopyCol:   db.GetCollection(cfg.DBName, "copies"),
		MemberCol: db.GetCollection(cfg.DBName, "members"),
		LoanCol:   db.GetCollection(cfg.DBName, "loans"),
		Config:    struct{ FineRate float64 }{FineRate: cfg.FineRate},
	}

	r.HandleFunc("/admin/metrics", metricsHandler.GetMetrics).Methods("GET")

	var server = http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Println("Server starting on port", cfg.Port)
		log.Fatal(server.ListenAndServe())
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}
	log.Println("Server shut down.")
}

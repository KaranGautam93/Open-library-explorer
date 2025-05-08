package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                       string
	MongoURI                   string
	DBName                     string
	JWTSecret                  string
	FineRate                   float64
	UserId                     string
	UserName                   string
	UserPassword               string
	PremiumMembersRenewalDays  int
	StandardMembersRenewalDays int
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	var fineRate float64

	if val := os.Getenv("FINE_RATE"); val != "" {
		_, err := fmt.Sscanf(val, "%f", &fineRate)
		if err != nil {
			log.Fatalf("Invalid FINE_RATE: %v", err)
		}
	}

	var premiumMemberRenewalDays, standardMemberRenewalDays int

	fmt.Sscanf(os.Getenv("PREMIUM_MEMBER_RENEWAL_DAYS"), "%d", &premiumMemberRenewalDays)
	fmt.Sscanf(os.Getenv("PREMIUM_MEMBER_RENEWAL_DAYS"), "%d", &standardMemberRenewalDays)

	return Config{
		Port:                       os.Getenv("PORT"),
		MongoURI:                   os.Getenv("MONGO_URI"),
		DBName:                     os.Getenv("DB_NAME"),
		JWTSecret:                  os.Getenv("JWT_SECRET"),
		FineRate:                   fineRate,
		UserId:                     os.Getenv("HARD_CODED_USER_ID"),
		UserName:                   os.Getenv("HARD_CODED_USER_NAME"),
		UserPassword:               os.Getenv("HARD_CODED_USER_PASSWORD"),
		PremiumMembersRenewalDays:  premiumMemberRenewalDays,
		StandardMembersRenewalDays: standardMemberRenewalDays,
	}
}

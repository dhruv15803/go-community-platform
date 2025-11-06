package main

import (
	"errors"
	"flag"
	"github.com/dhruv15803/go-community-platform/internal/database"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/dhruv15803/go-community-platform/scripts"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

type dbConfig struct {
	dbConnStr       string
	maxOpenConns    int
	maxIdleConns    int
	maxConnLifetime time.Duration
	maxConnIdleTime time.Duration
}

type config struct {
	dbConfig
}

func loadConfig() (*config, error) {

	godotenv.Load()

	dbConnStr := os.Getenv("POSTGRES_DB_CONN")

	if dbConnStr == "" {
		return nil, errors.New("$POSTGRES_DB_CONN env not set")
	}

	return &config{
		dbConfig: dbConfig{
			dbConnStr:       dbConnStr,
			maxOpenConns:    25,
			maxIdleConns:    10,
			maxConnLifetime: time.Hour,
			maxConnIdleTime: time.Minute * 10,
		},
	}, nil
}

func main() {

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	db, err := database.NewPostgresConn(cfg.dbConfig.dbConnStr, cfg.dbConfig.maxOpenConns, cfg.dbConfig.maxIdleConns, cfg.dbConfig.maxConnLifetime, cfg.dbConfig.maxConnIdleTime).Connect()
	if err != nil {
		log.Fatalf("Error connecting to postgres: %v\n", err)
	}
	defer db.Close()

	storage := storage.NewStorage(db)

	scripts := scripts.NewScripts(storage)

	userEmailPtr := flag.String("email", "", "user email")
	userPasswordPtr := flag.String("password", "", "user password")
	flag.Parse()

	userEmail := *userEmailPtr
	userPassword := *userPasswordPtr

	user, err := scripts.CreateTestUser(userEmail, userPassword)
	if err != nil {
		log.Fatalf("Error creating test user: %v\n", err)
	}

	log.Println("created test user: ", user)
}

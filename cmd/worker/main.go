package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dhruv15803/go-community-platform/internal/mailer"
	"github.com/dhruv15803/go-community-platform/internal/redis"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"time"
)

type mailerConfig struct {
	host     string
	port     int
	username string
	password string
}

type redisConfig struct {
	addr     string
	password string
	db       int
}

type config struct {
	redisConfig  redisConfig
	mailerConfig mailerConfig
}

func loadConfig() (*config, error) {

	godotenv.Load()

	mailerHost := os.Getenv("MAILER_HOST")
	mailerPortStr := os.Getenv("MAILER_PORT")
	mailerUsername := os.Getenv("MAILER_USERNAME")
	mailerPassword := os.Getenv("MAILER_PASSWORD")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	mailerPort, err := strconv.Atoi(mailerPortStr)
	if err != nil {
		return nil, errors.New("$MAILER_PORT should be an integer")
	}

	if mailerHost == "" || mailerPortStr == "" || mailerUsername == "" || mailerPassword == "" {
		return nil, errors.New("$MAILER env variables not set")
	}

	if redisAddr == "" {
		return nil, errors.New("$REDIS_ADDR not set")
	}

	return &config{
		redisConfig: redisConfig{
			addr:     redisAddr,
			password: redisPassword,
			db:       0, //default db
		},
		mailerConfig: mailerConfig{
			host:     mailerHost,
			port:     mailerPort,
			username: mailerUsername,
			password: mailerPassword,
		},
	}, nil
}

func main() {

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	mailer := mailer.NewMailer(cfg.mailerConfig.host, cfg.mailerConfig.port, cfg.mailerConfig.username, cfg.mailerConfig.password)

	rdb, err := redis.NewRedisConn(cfg.redisConfig.addr, cfg.redisConfig.password, cfg.redisConfig.db).Connect()
	if err != nil {
		log.Fatalf("Error connecting to redis: %v\n", err)
	}
	defer rdb.Close()

	log.Println("Connected to redis")

	for {

		type VerificationMailJob struct {
			FromEmail         string `json:"from_email"`
			ToEmail           string `json:"to_email"`
			UserId            int    `json:"user_id"`
			Subject           string `json:"subject"`
			EmailTemplatePath string `json:"email_template_path"`
			Token             string `json:"token"`
		}

		result, err := rdb.BRPop(context.Background(), 0, "queue:email").Result()
		if err != nil {
			log.Printf("Error retrieving queue element: %v\n", err)
			continue
		}
		var verificationMailJob VerificationMailJob
		verificationMailJobStr := result[1]
		if err := json.Unmarshal([]byte(verificationMailJobStr), &verificationMailJob); err != nil {
			log.Printf("Error unmarshalling mail job: %v\n", err)
			//	push popped job to a dlq
			return
		}

		maxEmailRetries := 3
		isEmailSent := false
		for i := 0; i < maxEmailRetries; i++ {

			if err := mailer.SendVerificationMail(verificationMailJob.FromEmail, verificationMailJob.ToEmail, verificationMailJob.Subject, verificationMailJob.Token, verificationMailJob.EmailTemplatePath); err != nil {
				log.Printf("Error sending verification mail, attempt %d : %v\n", i+1, err)
				continue
			}
			isEmailSent = true
			break
		}

		if !isEmailSent {

			log.Printf("Error sending verification mail, attempt %d : %v\n", maxEmailRetries, err)

			type VerificationMailJobFailureDetail struct {
				UserId    int                 `json:"user_id"`
				UserEmail string              `json:"user_email"`
				TimeStamp time.Time           `json:"timestamp"`
				Job       VerificationMailJob `json:"job"`
			}

			jobFailure := VerificationMailJobFailureDetail{
				UserId:    verificationMailJob.UserId,
				UserEmail: verificationMailJob.ToEmail,
				TimeStamp: time.Now(),
				Job:       verificationMailJob,
			}

			jobFailureJson, _ := json.Marshal(jobFailure)
			_ = rdb.LPush(context.Background(), "queue:email:dlq", string(jobFailureJson)).Err()

			continue
		}

		log.Printf("Email sent successfully to %s\n", verificationMailJob.ToEmail)
	}

}

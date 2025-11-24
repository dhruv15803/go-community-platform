package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dhruv15803/go-community-platform/internal/s3"
	"github.com/rs/cors"

	"github.com/dhruv15803/go-community-platform/internal/database"
	"github.com/dhruv15803/go-community-platform/internal/handlers"
	"github.com/dhruv15803/go-community-platform/internal/redis"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type redisConfig struct {
	addr     string
	password string
	db       int
}

type mailerConfig struct {
	host     string
	port     int
	username string
	password string
}

type dbConfig struct {
	dbConnStr       string
	maxOpenConns    int
	maxIdleConns    int
	maxConnLifetime time.Duration
	maxConnIdleTime time.Duration
}

type config struct {
	addr                string
	readRequestTimeout  time.Duration
	writeRequestTimeout time.Duration
	clientUrl           string
	dbConfig            dbConfig
	mailerConfig        mailerConfig
	redisConfig         redisConfig
}

func loadConfig() (*config, error) {

	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	port := os.Getenv("PORT")
	dbConnStr := os.Getenv("POSTGRES_DB_CONN")
	clientUrl := os.Getenv("CLIENT_URL")
	mailerHost := os.Getenv("MAILER_HOST")
	mailerPortStr := os.Getenv("MAILER_PORT")
	mailerUsername := os.Getenv("MAILER_USERNAME")
	mailerPassword := os.Getenv("MAILER_PASSWORD")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	if port == "" || dbConnStr == "" || clientUrl == "" {
		return nil, errors.New("$PORT or $POSTGRES_DB_CONN or $CLIENT_URL not set")
	}

	if mailerHost == "" || mailerPortStr == "" {
		return nil, errors.New("$MAILER_HOST or $MAILER_PORT not set")
	}

	if mailerUsername == "" || mailerPassword == "" {
		return nil, errors.New("$MAILER_USERNAME or $MAILER_PASSWORD not set")
	}

	if redisAddr == "" {
		return nil, errors.New("$REDIS_ADDR not set")
	}

	mailerPort, err := strconv.Atoi(mailerPortStr)
	if err != nil {
		return nil, errors.New("$MAILER_PORT should be an integer")
	}

	return &config{
		addr:                ":" + port,
		readRequestTimeout:  time.Second * 15,
		writeRequestTimeout: time.Second * 15,
		clientUrl:           clientUrl,
		dbConfig: dbConfig{
			dbConnStr:       dbConnStr,
			maxOpenConns:    25,
			maxIdleConns:    10,
			maxConnLifetime: time.Hour,
			maxConnIdleTime: time.Minute * 10,
		},
		mailerConfig: mailerConfig{
			host:     mailerHost,
			port:     mailerPort,
			username: mailerUsername,
			password: mailerPassword,
		},
		redisConfig: redisConfig{
			addr:     redisAddr,
			password: redisPassword,
			db:       0, // default db
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
		log.Fatalf("Error connecting to postgres database: %v\n", err)
	}
	defer db.Close()

	log.Println("connected to postgres database")

	rdb, err := redis.NewRedisConn(cfg.redisConfig.addr, cfg.redisConfig.password, cfg.redisConfig.db).Connect()
	if err != nil {
		log.Fatalf("Error connecting to redis: %v\n", err)
	}
	log.Println("connected to redis")
	defer rdb.Close()

	s3Client, err := s3.NewS3().NewClient()
	if err != nil {
		log.Fatalf("Error connecting to s3: %v\n", err)
	}

	storage := storage.NewStorage(db)
	handler := handlers.NewHandler(storage, rdb, s3Client)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
		AllowCredentials: true,
	})

	r := chi.NewRouter()

	r.Use(c.Handler)
	r.Use(middleware.Logger)
	r.Route("/api", func(r chi.Router) {

		r.Get("/health", handler.HealthCheckHandler)

		r.Route("/file", func(r chi.Router) {
			r.Use(handler.AuthMiddleware)
			r.Post("/upload", handler.UserImageFileUploadHandler)
		})

		r.Route("/auth", func(r chi.Router) {

			r.Post("/register", handler.RegisterUserHandler)
			r.Put("/activate/{token}", handler.ActivateUserHandler)
			r.Post("/login", handler.LoginUserHandler)
			r.With(handler.AuthMiddleware).Get("/user", handler.GetAuthUserHandler)
			r.With(handler.AuthMiddleware).Get("/logout", handler.LogoutHandler)

		})

		r.Route("/topics", func(r chi.Router) {

			r.Get("/", handler.GetTopicsHandler) // get topics by alphabetical order and get 10 or 20 at a time(page wise)

			r.Group(func(r chi.Router) {
				r.Use(handler.AuthMiddleware)
				r.Use(handler.AdminMiddleware)
				r.Post("/", handler.CreateTopicHandler)
				r.Delete("/{topicId}", handler.DeleteTopicHandler)
				r.Put("/{topicId}", handler.UpdateTopicHandler)
			})

		})

		r.Route("/topic-preferences", func(r chi.Router) {

			r.Use(handler.AuthMiddleware)
			r.Post("/", handler.CreateTopicPreferencesHandler) // add topics[] as authenticated user preference
			r.Delete("/{topicId}", handler.DeleteTopicPreferenceHandler)
			r.Get("/", handler.GetTopicPreferencesHandler)

		})

		r.Route("/communities", func(r chi.Router) {

			r.Get("/{communityId}", handler.GetCommunityHandler)

			r.Group(func(r chi.Router) {
				r.Use(handler.AuthMiddleware)
				r.Get("/recommended", handler.GetRecommendedCommunitiesHandler)
				r.Post("/", handler.CreateCommunityHandler)
				r.Post("/{communityId}/join", handler.ToggleJoinCommunityHandler)
				r.Get("/{communityId}/members", handler.GetCommunityMembersHandler)
			})

			// join community route

			r.Route("/{communityId}/posts", func(r chi.Router) {

				r.Get("/", handler.GetCommunityPostsHandler)

				r.Group(func(r chi.Router) {
					r.Use(handler.AuthMiddleware)
					r.Delete("/{postId}", handler.DeleteCommunityPostHandler)
					r.Post("/", handler.CreateCommunityPostHandler) // create a post in community
				})

			})
		})

		r.Route("/posts", func(r chi.Router) {

			r.With(handler.AuthMiddleware).Get("/feed", handler.GetUserPostsFeedHandler) // feed for logged in users consisting of posts of top communities they are a part of
			r.Get("/explore", handler.GetPostsFeedHandler)                               // for all users (no personalized according to joined communities)

			r.Group(func(r chi.Router) {

				r.Route("/{postId}", func(r chi.Router) {

					r.Group(func(r chi.Router) {
						r.Use(handler.AuthMiddleware)
						r.Post("/like", handler.TogglePostLikeHandler)
						r.Post("/bookmark", handler.TogglePostBookmarkHandler)
					})

					r.Route("/comments", func(r chi.Router) {

						r.Get("/", handler.GetPostCommentsHandler)
						r.Get("/{commentId}/replies", handler.GetCommentRepliesHandler)

						r.Group(func(r chi.Router) {
							r.Use(handler.AuthMiddleware)
							r.Post("/", handler.CreatePostCommentHandler)
							r.Delete("/{commentId}", handler.DeletePostCommentHandler)
						})

					})
				})
			})
		})

		r.Route("/comments", func(r chi.Router) {
			r.Use(handler.AuthMiddleware)
			r.Delete("/{commentId}", handler.DeletePostCommentHandler)
			r.Post("/{commentId}/like", handler.ToggleCommentLikeHandler)
		})

		r.Route("/users", func(r chi.Router) {
			// create a put handler to update authenticated user's username
			r.Use(handler.AuthMiddleware)
			r.Patch("/me/username", handler.UpdateUsernameHandler)
		})

	})

	server := http.Server{
		Addr:         cfg.addr,
		Handler:      r,
		ReadTimeout:  cfg.readRequestTimeout,
		WriteTimeout: cfg.writeRequestTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}

}

package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/dhruv15803/go-community-platform/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var (
	JWT_SECRET = []byte(os.Getenv("JWT_SECRET"))
	AuthUserId = "AuthUserId"
)

func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {

	var registerUserPayload RegisterUserRequest

	if err := readJSON(r, &registerUserPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userEmail := strings.ToLower(strings.TrimSpace(registerUserPayload.Email))
	userPlainTextPassword := strings.TrimSpace(registerUserPayload.Password)

	if userEmail == "" || userPlainTextPassword == "" {
		writeJSONError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	if !utils.IsEmailValid(userEmail) {
		writeJSONError(w, "invalid email", http.StatusBadRequest)
		return
	}

	if !utils.IsPasswordStrong(userPlainTextPassword) {
		writeJSONError(w, "password is weak", http.StatusBadRequest)
		return
	}

	//	check if a user already exists with this email (verified user)
	existingUser, err := h.storage.Users.GetVerifiedUserByEmail(userEmail)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("failed query GetVerifiedUserByEmail: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		writeJSONError(w, "user already exists", http.StatusBadRequest)
		return
	}

	// user with this email does not exist
	// create a new  entry for this user
	hashedPasswordByteArr, err := bcrypt.GenerateFromPassword([]byte(userPlainTextPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	hashedPassword := string(hashedPasswordByteArr)

	//	create a entry in users(table) and a user_invitation entry as well
	plainTextToken := generateToken(32)
	hashedToken := hashPlainTextToken(plainTextToken)

	//	create user and user invitation
	invitationExpirationTime := time.Now().Add(time.Minute * 30)
	user, err := h.storage.Users.CreateUserAndInvitation(userEmail, hashedPassword, hashedToken, invitationExpirationTime)
	if err != nil {
		log.Printf("failed createUserAndInvitation: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type VerificationMailJob struct {
		FromEmail         string `json:"from_email"`
		ToEmail           string `json:"to_email"`
		UserId            int    `json:"user_id"`
		Subject           string `json:"subject"`
		EmailTemplatePath string `json:"email_template_path"`
		Token             string `json:"token"`
	}

	verificationMailJob := VerificationMailJob{
		FromEmail:         os.Getenv("MAILER_USERNAME"),
		ToEmail:           user.Email,
		UserId:            user.Id,
		Subject:           "Verify your account",
		Token:             plainTextToken,
		EmailTemplatePath: "./templates/verification_mail.html",
	}

	verificationMailJobJson, err := json.Marshal(verificationMailJob)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	maxRetries := 3
	isJobPushSuccessful := false
	for i := 0; i < maxRetries; i++ {

		if err := h.rdb.LPush(context.Background(), "queue:email", string(verificationMailJobJson)).Err(); err != nil {
			log.Printf("failed to push email job into queue, attempty %d : %v\n", i+1, err)
			continue
		}

		isJobPushSuccessful = true
		break

	}
	if !isJobPushSuccessful {
		// include the user id for the user  that has been added to the application's db
		// but their mail didn't get sent, and at what time
		type VerificationMailJobFailureDetail struct {
			UserId    int                 `json:"user_id"`
			UserEmail string              `json:"user_email"`
			TimeStamp time.Time           `json:"timestamp"`
			Job       VerificationMailJob `json:"job"`
		}

		jobFailure := VerificationMailJobFailureDetail{
			UserId:    user.Id,
			UserEmail: user.Email,
			TimeStamp: time.Now(),
			Job:       verificationMailJob,
		}

		jobFailureJson, err := json.Marshal(jobFailure)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		_ = h.rdb.LPush(context.Background(), "queue:email:dlq", string(jobFailureJson)).Err()

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "Your account has been created. We are experiencing a temporary email delivery issue. You may recieve the verification email shortly"}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

	}

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user registered successfully", User: *user}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}

}

func (h *Handler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {

	plainTextToken := chi.URLParam(r, "token")
	log.Println("activate user ", plainTextToken)

	//	activate user associated with this token
	hashedToken := hashPlainTextToken(plainTextToken)

	// get user_invitation with this hashed token
	// get userId , update user with id=userId to is_verified=true
	updatedUser, err := h.storage.Users.ActivateUser(hashedToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "no invitation found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	claims := jwt.MapClaims{
		"sub": updatedUser.Id,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(JWT_SECRET)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "production" {
		sameSiteConfig = http.SameSiteNoneMode
	} else {
		sameSiteConfig = http.SameSiteLaxMode
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    tokenStr,
		HttpOnly: true,
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		Path:     "/",
		MaxAge:   60 * 60 * 24,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user verified successfully", User: *updatedUser}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}

}

func (h *Handler) LoginUserHandler(w http.ResponseWriter, r *http.Request) {

	var loginUserPayload LoginUserRequest

	if err := readJSON(r, &loginUserPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userEmail := strings.ToLower(strings.TrimSpace(loginUserPayload.Email))
	userPassword := strings.TrimSpace(loginUserPayload.Password)

	if userEmail == "" || userPassword == "" {
		writeJSONError(w, "email and password required", http.StatusBadRequest)
		return
	}

	user, err := h.storage.Users.GetVerifiedUserByEmail(userEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "invalid email or password", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userPassword)); err != nil {
		writeJSONError(w, "invalid email or password", http.StatusBadRequest)
		return
	}

	//  email and password correct

	claims := jwt.MapClaims{
		"sub": user.Id,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(JWT_SECRET)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "production" {
		sameSiteConfig = http.SameSiteNoneMode
	} else {
		sameSiteConfig = http.SameSiteLaxMode
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    tokenStr,
		HttpOnly: true,
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		Path:     "/",
		MaxAge:   60 * 60 * 24,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user logged in", User: *user}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("auth_token")
		if err != nil {
			writeJSONError(w, "auth token not found", http.StatusBadRequest)
			return
		}

		tokenStr := cookie.Value

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {

			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}

			return JWT_SECRET, nil
		})

		if err != nil {
			log.Printf("failed to parse token:- %v\n", err)
			writeJSONError(w, "invalid token", http.StatusBadRequest)
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			expirationTimeFloat := claims["exp"].(float64)
			expirationTime := int(expirationTimeFloat)
			userIdFloat := claims["sub"].(float64)
			userId := int(userIdFloat)

			if time.Now().Unix() > int64(expirationTime) {
				writeJSONError(w, "auth token expired", http.StatusBadRequest)
				return
			}

			ctx := context.WithValue(r.Context(), AuthUserId, userId)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
	})
}

func (h *Handler) AdminMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//	admin middleware will only be used after auth middleware
		// user needs to be authenticated then checked for admin

		userId, ok := r.Context().Value(AuthUserId).(int)
		if !ok {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		user, err := h.storage.Users.GetUserById(userId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, "user not found", http.StatusNotFound)
				return
			} else {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		if user.Role != "admin" {
			writeJSONError(w, "user is not admin", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})

}

func (h *Handler) GetAuthUserHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	user, err := h.storage.Users.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	type Response struct {
		Success bool         `json:"success"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, User: *user}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	_, err := h.storage.Users.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// user entry exists

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "production" {
		sameSiteConfig = http.SameSiteNoneMode
	} else {
		sameSiteConfig = http.SameSiteLaxMode
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		HttpOnly: true,
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		Path:     "/",
		MaxAge:   0,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "logged out successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func generateToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	plainTextToken := hex.EncodeToString(b)
	return plainTextToken
}

func hashPlainTextToken(token string) string {
	hashedTokenByteArr := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashedTokenByteArr[:])
}

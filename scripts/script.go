package scripts

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/dhruv15803/go-community-platform/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type Scripts struct {
	storage *storage.Storage
}

func NewScripts(storage *storage.Storage) *Scripts {
	return &Scripts{
		storage: storage,
	}
}

func (s *Scripts) CreateAdminUser(email string, password string) (*storage.User, error) {

	userEmail := strings.ToLower(strings.TrimSpace(email))
	userPassword := strings.TrimSpace(password)

	if userEmail == "" || userPassword == "" {
		return nil, errors.New("email and password required")
	}

	if !utils.IsEmailValid(userEmail) {
		return nil, errors.New("invalid email")
	}

	if !utils.IsPasswordStrong(userPassword) {
		return nil, errors.New("weak password")
	}

	existingUser, err := s.storage.Users.GetVerifiedUserByEmail(userEmail)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get verified user by email: %v\n", err)
	}
	if existingUser != nil {
		return nil, errors.New("user already exists with this email")
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashedPassword := string(hashedPasswordBytes)

	user, err := s.storage.Users.CreateAdminUser(userEmail, hashedPassword)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Scripts) CreateTestUser(email string, password string) (*storage.User, error) {

	userEmail := strings.ToLower(strings.TrimSpace(email))
	userPassword := strings.TrimSpace(password)

	if userEmail == "" || userPassword == "" {
		return nil, errors.New("email and password required")
	}

	if !utils.IsEmailValid(userEmail) {
		return nil, errors.New("invalid email")
	}

	if !utils.IsPasswordStrong(userPassword) {
		return nil, errors.New("weak password")
	}

	existingUser, err := s.storage.Users.GetVerifiedUserByEmail(userEmail)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get verified user by email: %v\n", err)
	}
	if existingUser != nil {
		return nil, errors.New("user already exists with this email")
	}

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashedPassword := string(hashedPasswordBytes)

	user, err := s.storage.Users.CreateVerifiedUser(userEmail, hashedPassword)
	if err != nil {
		return nil, err
	}

	return user, nil
}

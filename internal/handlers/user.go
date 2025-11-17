package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/dhruv15803/go-community-platform/internal/storage"
)

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

func (h *Handler) UpdateUsernameHandler(w http.ResponseWriter, r *http.Request) {

	// authenticated endpoint

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

	var updateUsernamePayload UpdateUsernameRequest

	if err := readJSON(r, &updateUsernamePayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newUsername := strings.TrimSpace(updateUsernamePayload.Username)

	if newUsername == "" {
		writeJSONError(w, "username is required", http.StatusBadRequest)
		return
	}

	// check if user with username already exists

	existingUserWithUsername, err := h.storage.Users.GetUserByUsername(newUsername)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingUserWithUsername != nil {

		// a user with this username already exists

		if existingUserWithUsername.Id == user.Id {

			// give a success response

		} else {

			// error response

		}

	} else {

		// update user with this username

		newUser, err := h.storage.Users.UpdateUsernameById(user.Id, newUsername)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool         `json:"success"`
			Message string       `json:"message"`
			User    storage.User `json:"user"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "updated username successfully", User: *newUser}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

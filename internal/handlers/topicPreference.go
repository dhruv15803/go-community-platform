package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strconv"
)

const (
	MIN_USER_TOPIC_PREFERENCE = 3
	MAX_USER_TOPIC_PREFERENCE = 10
)

type Topic struct {
	Id        int    `json:"id"`
	TopicName string `json:"topic_name"`
}

type CreateTopicPreferencesRequest struct {
	Topics []Topic `json:"topics"`
}

func (h *Handler) CreateTopicPreferencesHandler(w http.ResponseWriter, r *http.Request) {

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

	// check user's existing topic preferences
	// if its greater than or equal to MAX then cannot proceed without removing existing preferences
	// else if , check that the no of unique topicIds trying to add does not exceed the total amount to MAX or more

	existingTopicPreferences, err := h.storage.UserTopicPreferences.GetUserTopicPreferences(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(existingTopicPreferences) == MAX_USER_TOPIC_PREFERENCE {
		writeJSONError(w, "cannot create more topic preferences", http.StatusBadRequest)
		return
	}

	var createTopicPreferencesPayload CreateTopicPreferencesRequest

	if err := readJSON(r, &createTopicPreferencesPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var topicIds []int // array of unique topicIds
	topics := createTopicPreferencesPayload.Topics

	//	extract unique topic Ids from topics

	for _, topic := range topics {
		if isArrayContainsElement(topicIds, topic.Id) {
			continue
		} else {
			topicIds = append(topicIds, topic.Id)
		}
	}

	// check that each topicId is a valid existing topic
	// if it is valid , check if user already has topic as preference
	// if yes, skip over it and add the rest of the topicIds to user preference
	var correctTopicIds []int // final topic Ids to add to user preference

	for _, topicId := range topicIds {

		topic, err := h.storage.Topics.GetTopicById(topicId)
		if err != nil {

			if errors.Is(err, sql.ErrNoRows) {

				writeJSONError(w, fmt.Sprintf("incorrect topic id %d\n", topicId), http.StatusBadRequest)
				return

			} else {

				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return

			}

		}

		existingTopicPreference, err := h.storage.UserTopicPreferences.GetUserTopicPreference(user.Id, topic.Id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if existingTopicPreference != nil {
			log.Printf("skipping existing user topic preference with topic id %d\n", topic.Id)
			continue
		}

		correctTopicIds = append(correctTopicIds, topic.Id)

	}

	// correct topic ids array is the array of topics that user hasn't selected/preffered

	// now check that if by adding this topics to user's interest, will it exceed the MAX_TOPIC_PREFERENCE
	if len(correctTopicIds)+len(existingTopicPreferences) < MIN_USER_TOPIC_PREFERENCE {
		writeJSONError(w, fmt.Sprintf("user should have atleast %d topic preferences\n", MIN_USER_TOPIC_PREFERENCE), http.StatusBadRequest)
		return
	}

	if len(correctTopicIds)+len(existingTopicPreferences) > MAX_USER_TOPIC_PREFERENCE {
		writeJSONError(w, fmt.Sprintf("user can have max %d topic preferences", MAX_USER_TOPIC_PREFERENCE), http.StatusBadRequest)
		return
	}

	userTopicPreferences, err := h.storage.UserTopicPreferences.CreateUserTopicPreferences(user.Id, correctTopicIds)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success              bool                          `json:"success"`
		Message              string                        `json:"message"`
		UserTopicPreferences []storage.UserTopicPreference `json:"user_topic_preferences"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "added topic preferences", UserTopicPreferences: userTopicPreferences}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) DeleteTopicPreferenceHandler(w http.ResponseWriter, r *http.Request) {

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

	topicId, err := strconv.Atoi(chi.URLParam(r, "topicId"))
	if err != nil {
		writeJSONError(w, "invalid request param topicId", http.StatusBadRequest)
		return
	}

	topic, err := h.storage.Topics.GetTopicById(topicId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "topic not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	topicPreference, err := h.storage.UserTopicPreferences.GetUserTopicPreference(user.Id, topic.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "topic preference not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if err = h.storage.UserTopicPreferences.DeleteUserTopicPreference(topicPreference.UserId, topicPreference.TopicId); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "deleted user topic preference successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}

}

func (h *Handler) GetTopicPreferencesHandler(w http.ResponseWriter, r *http.Request) {

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

	// get topics preferred by this user (no pagination required as a user's preffered topics will be capped to lets say 5 or 10 at all times)

	topics, err := h.storage.TopicQueryRepository.GetTopicsPrefferedByUser(user.Id)
	if err != nil {
		log.Printf("failed to get preffered topics: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool            `json:"success"`
		Topics  []storage.Topic `json:"topics"`
	}

	if err := writeJSON(w, Response{Success: true, Topics: topics}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func isArrayContainsElement[T int | string | float64 | float32 | bool](arr []T, element T) bool {

	for _, val := range arr {
		if val == element {
			return true
		}
	}

	return false
}

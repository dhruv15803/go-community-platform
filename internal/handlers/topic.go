package handlers

import (
	"database/sql"
	"errors"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type CreateTopicRequest struct {
	TopicName string `json:"topic_name"`
}

type UpdateTopicRequest struct {
	TopicName string `json:"topic_name"`
}

// admin route
func (h *Handler) CreateTopicHandler(w http.ResponseWriter, r *http.Request) {

	var createTopicPayload CreateTopicRequest

	if err := readJSON(r, &createTopicPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	topicName := strings.ToLower(strings.TrimSpace(createTopicPayload.TopicName))

	if topicName == "" {
		writeJSONError(w, "topic name is required", http.StatusBadRequest)
		return
	}

	existingTopic, err := h.storage.Topics.GetTopicByTopicName(topicName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingTopic != nil {
		writeJSONError(w, "topic already exists", http.StatusBadRequest)
		return
	}

	// create new topic
	topic, err := h.storage.Topics.CreateTopic(topicName)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool          `json:"success"`
		Message string        `json:"message"`
		Topic   storage.Topic `json:"topic"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "created topic", Topic: *topic}, http.StatusCreated); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) DeleteTopicHandler(w http.ResponseWriter, r *http.Request) {

	topicId, err := strconv.Atoi(chi.URLParam(r, "topicId"))
	if err != nil {
		writeJSONError(w, "invalid request param topicId", http.StatusBadRequest)
		return
	}

	topic, err := h.storage.Topics.GetTopicById(topicId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "topic not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	//	delete topic
	if err = h.storage.Topics.DeleteTopicById(topic.Id); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "deleted topic"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) UpdateTopicHandler(w http.ResponseWriter, r *http.Request) {

	topicId, err := strconv.Atoi(chi.URLParam(r, "topicId"))
	if err != nil {
		writeJSONError(w, "invalid request param topicId", http.StatusBadRequest)
		return
	}

	topic, err := h.storage.Topics.GetTopicById(topicId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "topic not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	var updateTopicPayload UpdateTopicRequest

	if err := readJSON(r, &updateTopicPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newTopicName := updateTopicPayload.TopicName
	if newTopicName == "" {
		writeJSONError(w, "topic name is required", http.StatusBadRequest)
		return
	}

	topicWithNewName, err := h.storage.Topics.GetTopicByTopicName(newTopicName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if topicWithNewName != nil {
		// topic with new name already exists
		if topicWithNewName.Id == topic.Id {
			// topic has already new name
			type Response struct {
				Success bool          `json:"success"`
				Message string        `json:"message"`
				Topic   storage.Topic `json:"topic"`
			}

			if err := writeJSON(w, Response{Success: true, Message: "topic updated successfully", Topic: *topicWithNewName}, http.StatusOK); err != nil {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
			}
			return
		} else {
			writeJSONError(w, "topic with new name already exists", http.StatusBadRequest)
			return
		}
	}

	updatedTopic, err := h.storage.Topics.UpdateTopicById(topic.Id, newTopicName)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool          `json:"success"`
		Message string        `json:"message"`
		Topic   storage.Topic `json:"topic"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "updated topic", Topic: *updatedTopic}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}

}

// get topics by alphabetical order and get 10 or 20 at a time(page wise)
func (h *Handler) GetTopicsHandler(w http.ResponseWriter, r *http.Request) {

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	topics, err := h.storage.Topics.GetTopics(skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalTopicsCount, err := h.storage.Topics.GetTopicsCount()
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalTopicsCount) / float64(limit)))

	type Response struct {
		Success   bool            `json:"success"`
		Topics    []storage.Topic `json:"topics"`
		NoOfPages int             `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Topics: topics, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

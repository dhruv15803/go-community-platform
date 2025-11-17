package handlers

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
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

func (h *Handler) GetTopicsHandler(w http.ResponseWriter, r *http.Request) {

	var page int
	var limit int
	var search string
	var err error

	if r.URL.Query().Get("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			writeJSONError(w, "invalid query params page", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("limit") == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			writeJSONError(w, "invalid query params limit", http.StatusBadRequest)
			return
		}
	}

	search = r.URL.Query().Get("search")

	skip := page*limit - limit

	// if search === "" -> get all topics (no filtration) else filter by topic_title

	topics, err := h.storage.Topics.GetTopics(skip, limit, search)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalTopicsCount, err := h.storage.Topics.GetTopicsCount(search)
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

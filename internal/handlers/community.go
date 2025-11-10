package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
)

type CreateCommunityRequest struct {
	CommunityName        string  `json:"community_name"`
	CommunityDescription string  `json:"community_description"`
	CommunityImage       string  `json:"community_image"`
	CommunityTopics      []Topic `json:"community_topics"`
}

const (
	MIN_COMMUNITY_TOPICS = 1
	MAX_COMMUNITY_TOPICS = 3
)

func (h *Handler) CreateCommunityHandler(w http.ResponseWriter, r *http.Request) {

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

	var createCommunityPayload CreateCommunityRequest

	if err := readJSON(r, &createCommunityPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	communityName := strings.TrimSpace(createCommunityPayload.CommunityName)
	communityDescription := strings.TrimSpace(createCommunityPayload.CommunityDescription)
	communityImageUrl := createCommunityPayload.CommunityImage
	communityTopics := createCommunityPayload.CommunityTopics

	if communityName == "" {
		writeJSONError(w, "community name is required", http.StatusBadRequest)
		return
	}

	// check if a community already exists with communityName
	existingCommunity, err := h.storage.Communities.GetCommunityByName(communityName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingCommunity != nil {
		writeJSONError(w, "community already exists", http.StatusBadRequest)
		return
	}

	if len(communityTopics) < MIN_COMMUNITY_TOPICS {
		writeJSONError(w, "community topics is required", http.StatusBadRequest)
		return
	}

	if len(communityTopics) > MAX_COMMUNITY_TOPICS {
		writeJSONError(w, fmt.Sprintf("community cannot have more than %d topics\n", MAX_COMMUNITY_TOPICS), http.StatusBadRequest)
		return
	}

	var uniqueTopicIds []int
	for _, topic := range communityTopics {

		if !isArrayContainsElement(uniqueTopicIds, topic.Id) {

			// check if valid topic id
			topic, err := h.storage.Topics.GetTopicById(topic.Id)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeJSONError(w, "topic not found", http.StatusBadRequest)
					return
				} else {
					writeJSONError(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}

			uniqueTopicIds = append(uniqueTopicIds, topic.Id)
		} else {
			continue
		}

	}

	// create new community with community topics

	communityWithTopics, err := h.storage.Communities.CreateCommunityWithTopics(communityName, communityDescription, communityImageUrl, user.Id, uniqueTopicIds)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success   bool                        `json:"success"`
		Message   string                      `json:"message"`
		Community storage.CommunityWithTopics `json:"community"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "created community successfully", Community: *communityWithTopics}, http.StatusCreated); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)

	}

}

// toggle join community handler (if already joined by user, remove user from community)
func (h *Handler) ToggleJoinCommunityHandler(w http.ResponseWriter, r *http.Request) {

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

	communityId, err := strconv.Atoi(chi.URLParam(r, "communityId"))
	if err != nil {
		writeJSONError(w, "invalid request param communityId", http.StatusBadRequest)
		return
	}

	community, err := h.storage.Communities.GetCommunityById(communityId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "community not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if user.Id == community.CommunityOwnerId {
		writeJSONError(w, "owner is already part of community", http.StatusBadRequest)
		return
	}

	isPartOfCommunity, err := h.storage.Communities.CheckCommunityForUser(user.Id, community.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if isPartOfCommunity {

		// remove user from community

		if err := h.storage.Communities.LeaveCommunity(user.Id, community.Id); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "left community successfully"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

		}

	} else {

		userCommunity, err := h.storage.Communities.JoinCommunity(user.Id, community.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success       bool                  `json:"success"`
			Message       string                `json:"message"`
			UserCommunity storage.UserCommunity `json:"user_community"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "joined community successfully", UserCommunity: *userCommunity}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

		}

	}
}

// who can access this community members list (owner) || (member)
func (h *Handler) GetCommunityMembersHandler(w http.ResponseWriter, r *http.Request) {

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

	communityId, err := strconv.Atoi(chi.URLParam(r, "communityId"))
	if err != nil {
		writeJSONError(w, "invalid request param communityId", http.StatusBadRequest)
		return
	}

	community, err := h.storage.Communities.GetCommunityById(communityId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "community not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	var page int
	var limit int
	if r.URL.Query().Get("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			writeJSONError(w, "invalid query param page", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("limit") == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
			return
		}
	}

	skip := page*limit - limit

	isPartOfCommunity, _ := h.storage.Communities.CheckCommunityForUser(user.Id, community.Id)
	if user.Id != community.CommunityOwnerId && !isPartOfCommunity {
		writeJSONError(w, "cannot view community members", http.StatusForbidden)
		return
	}

	members, err := h.storage.Communities.GetCommunityMembers(community.Id, skip, limit)
	if err != nil {
		log.Printf("failed to get community members: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	totalMembersCount, err := h.storage.Communities.GetTotalCommunityMembersCount(community.Id)
	if err != nil {
		log.Printf("failed to get total community members count: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	noOfPages := math.Ceil(float64(totalMembersCount) / float64(limit))

	type Response struct {
		Success   bool           `json:"success"`
		Members   []storage.User `json:"members"`
		NoOfPages int            `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Members: members, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}
}

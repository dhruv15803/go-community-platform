package handlers

import (
	"database/sql"
	"errors"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type CreatePostRequest struct {
	PostTitle     string   `json:"post_title"`
	PostContent   string   `json:"post_content"`
	PostImageUrls []string `json:"post_image_urls"`
}

// create community post handler
func (h *Handler) CreateCommunityPostHandler(w http.ResponseWriter, r *http.Request) {

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

	// check if user is owner or part of the community
	isPartOfCommunity, err := h.storage.Communities.CheckCommunityForUser(user.Id, community.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !isPartOfCommunity && user.Id != community.CommunityOwnerId {
		writeJSONError(w, "user cannot create post", http.StatusForbidden)
		return
	}

	var createPostPayload CreatePostRequest

	if err := readJSON(r, &createPostPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	postTitle := strings.TrimSpace(createPostPayload.PostTitle)
	postContent := strings.TrimSpace(createPostPayload.PostContent)
	postImageUrls := createPostPayload.PostImageUrls

	if postTitle == "" || postContent == "" {
		writeJSONError(w, "title and content required", http.StatusBadRequest)
		return
	}

	if len(postImageUrls) == 0 {

		post, err := h.storage.Posts.CreatePost(postTitle, postContent, user.Id, community.Id)
		if err != nil {
			log.Printf("error creating post: %v\n", err)
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool         `json:"success"`
			Message string       `json:"message"`
			Post    storage.Post `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "post created successfully", Post: *post}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	} else {

		postWithImages, err := h.storage.Posts.CreatePostWithImages(postTitle, postContent, user.Id, community.Id, postImageUrls)
		if err != nil {
			log.Printf("error creating post: %v\n", err)
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool                   `json:"success"`
			Message string                 `json:"message"`
			Post    storage.PostWithImages `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "post created successfully", Post: *postWithImages}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func (h *Handler) DeleteCommunityPostHandler(w http.ResponseWriter, r *http.Request) {

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

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postId", http.StatusBadRequest)
		return
	}

	post, err := h.storage.Posts.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// post has to be part of community , if yes then delete only if the user is community owner or post owner

	if post.PostCommunityId != community.Id {
		writeJSONError(w, "post is not part of community", http.StatusBadRequest)
		return
	}

	if user.Id != post.PostOwnerId && user.Id != community.CommunityOwnerId {
		writeJSONError(w, "user unauthorized to delete community post", http.StatusForbidden)
		return
	}

	if err = h.storage.Posts.DeletePostById(postId); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "successfully deleted post"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

// toggle like post handler
func (h *Handler) TogglePostLikeHandler(w http.ResponseWriter, r *http.Request) {

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

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postIod", http.StatusBadRequest)
		return
	}

	post, err := h.storage.Posts.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// check if post like by user already exists
	// yes -> remove like
	//	 no -> creat like

	isPostLiked, _ := h.storage.Posts.CheckPostLike(user.Id, post.Id)

	if isPostLiked {

		//	 remove post like

		if err := h.storage.Posts.RemovePostLike(user.Id, post.Id); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err = writeJSON(w, Response{Success: true, Message: "removed like"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	} else {
		postLike, err := h.storage.Posts.CreatePostLike(user.Id, post.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success  bool             `json:"success"`
			Message  string           `json:"message"`
			PostLike storage.PostLike `json:"post_like"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "liked post", PostLike: *postLike}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	}

}

func (h *Handler) TogglePostBookmarkHandler(w http.ResponseWriter, r *http.Request) {

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

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postIod", http.StatusBadRequest)
		return
	}

	post, err := h.storage.Posts.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	isPostBookmarked, _ := h.storage.Posts.CheckPostBookmark(user.Id, post.Id)

	if isPostBookmarked {

		if err := h.storage.Posts.RemovePostBookmark(user.Id, post.Id); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "removed bookmark"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	} else {

		postBookmark, err := h.storage.Posts.CreatePostBookmark(user.Id, post.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success      bool                 `json:"success"`
			Message      string               `json:"message"`
			PostBookmark storage.PostBookmark `json:"post_bookmark"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "bookmarked post", PostBookmark: *postBookmark}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	}
}

// ?page=1&limit=10&sortBy="hot"
// no auth required
// ?page=1&limit=10&sortBy="new"
// ?page=1&limit=10&sortBy="top"&search="goasdkajsda"

func (h *Handler) GetCommunityPostsHandler(w http.ResponseWriter, r *http.Request) {

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
	var search string
	var sortBy storage.SortByStr

	if r.URL.Query().Get("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			writeJSONError(w, "invalid request param page", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("limit") == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			writeJSONError(w, "invalid request param limit", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("sortBy") == "" {
		sortBy = storage.SortByRelevance
	} else {
		sortBy = storage.SortByStr(r.URL.Query().Get("sortBy"))
	}
	search = r.URL.Query().Get("search")

	if sortBy != storage.SortByNewest && sortBy != storage.SortByTop && sortBy != storage.SortByRelevance {
		writeJSONError(w, "invalid request param sortBy", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	posts, err := h.storage.Posts.GetCommunityPosts(community.Id, skip, limit, sortBy, search)
	if err != nil {
		log.Printf("failed to get posts: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostsCount, err := h.storage.Posts.GetCommunityPostsCount(community.Id, search)
	if err != nil {
		log.Printf("failed to get posts count: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalPostsCount) / float64(limit)))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetUserPostsFeedHandler(w http.ResponseWriter, r *http.Request) {

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

	var page int
	var limit int
	var sortBy storage.SortByStr

	if r.URL.Query().Get("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			writeJSONError(w, "invalid request param page", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("limit") == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			writeJSONError(w, "invalid request param limit", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("sortBy") == "" {
		sortBy = storage.SortByRelevance
	} else {
		sortBy = storage.SortByStr(r.URL.Query().Get("sortBy"))
	}

	skip := page*limit - limit
	//fetchPostsFromTopNCommunitiesThatUserJoinedByNoOfMembers
	n := 3
	posts, err := h.storage.Posts.GetUserPostsFeed(user.Id, n, skip, limit, sortBy)
	if err != nil {
		log.Printf("failed to get posts: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostsCount, err := h.storage.Posts.GetUserPostsFeedCount(user.Id, n)
	if err != nil {
		log.Printf("failed to get posts count: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalPostsCount) / float64(limit)))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetPostsFeedHandler(w http.ResponseWriter, r *http.Request) {

	var page int
	var limit int
	var sortBy storage.SortByStr
	var err error

	if r.URL.Query().Get("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			writeJSONError(w, "invalid request param page", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("limit") == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			writeJSONError(w, "invalid request param limit", http.StatusBadRequest)
			return
		}
	}

	if r.URL.Query().Get("sortBy") == "" {
		sortBy = storage.SortByRelevance
	} else {
		sortBy = storage.SortByStr(r.URL.Query().Get("sortBy"))
	}

	skip := page*limit - limit
	n := 3 // post from  top 3 communities of the application
	posts, err := h.storage.Posts.GetPostsFeed(n, skip, limit, sortBy)
	if err != nil {
		log.Printf("failed to get posts: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostsCount, err := h.storage.Posts.GetPostsFeedCount(n)
	if err != nil {
		log.Printf("failed to get posts count: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalPostsCount) / float64(limit)))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

}

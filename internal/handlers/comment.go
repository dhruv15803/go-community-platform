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

type CreatePostCommentRequest struct {
	CommentContent  string `json:"comment_content"`
	ParentCommentId *int   `json:"parent_comment_id"`
}

// /:postId
func (h *Handler) CreatePostCommentHandler(w http.ResponseWriter, r *http.Request) {

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

	var createPostCommentPayload CreatePostCommentRequest

	if err := readJSON(r, &createPostCommentPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	commentContent := strings.TrimSpace(createPostCommentPayload.CommentContent)
	parentCommentId := createPostCommentPayload.ParentCommentId
	isChildComment := parentCommentId != nil

	if isChildComment {

		// parent comment needs to have post_id = post.Id
		parentComment, err := h.storage.PostComments.GetPostCommentById(*parentCommentId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, "parent comment not found", http.StatusNotFound)
				return
			} else {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		if parentComment.PostId != post.Id {
			writeJSONError(w, "parent comment does not belong to post", http.StatusBadRequest)
			return
		}

		postChildComment, err := h.storage.PostComments.CreateChildPostComment(commentContent, user.Id, post.Id, parentComment.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success     bool                `json:"success"`
			Message     string              `json:"message"`
			PostComment storage.PostComment `json:"post_comment"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "created post comment", PostComment: *postChildComment}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

		}

	} else {

		// create normal comment
		postComment, err := h.storage.PostComments.CreatePostComment(commentContent, user.Id, post.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success     bool                `json:"success"`
			Message     string              `json:"message"`
			PostComment storage.PostComment `json:"post_comment"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "created post comment", PostComment: *postComment}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

		}
	}
}

func (h *Handler) DeletePostCommentHandler(w http.ResponseWriter, r *http.Request) {

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

	commentId, err := strconv.Atoi(chi.URLParam(r, "commentId"))
	if err != nil {
		writeJSONError(w, "invalid request param commentId", http.StatusBadRequest)
		return
	}

	comment, err := h.storage.PostComments.GetPostCommentById(commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post comment not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if user.Id != comment.CommentOwnerId {
		writeJSONError(w, "user not authorized to delete comment", http.StatusUnauthorized)
		return
	}

	if err = h.storage.PostComments.DeletePostCommentById(comment.Id); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "deleted comment successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) ToggleCommentLikeHandler(w http.ResponseWriter, r *http.Request) {

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

	commentId, err := strconv.Atoi(chi.URLParam(r, "commentId"))
	if err != nil {
		writeJSONError(w, "invalid request param commentId", http.StatusBadRequest)
		return
	}

	comment, err := h.storage.PostComments.GetPostCommentById(commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post comment not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	isCommentLiked, _ := h.storage.PostComments.CheckCommentLike(user.Id, comment.Id)

	if isCommentLiked {

		if err := h.storage.PostComments.RemoveCommentLike(user.Id, comment.Id); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "comment like removed"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	} else {

		postCommentLike, err := h.storage.PostComments.CreateCommentLike(user.Id, comment.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

			return
		}

		type Response struct {
			Success         bool                    `json:"success"`
			Message         string                  `json:"message"`
			PostCommentLike storage.PostCommentLike `json:"post_comment_like"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "comment liked", PostCommentLike: *postCommentLike}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	}

}

// no auth required to view post comments
func (h *Handler) GetPostCommentsHandler(w http.ResponseWriter, r *http.Request) {

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

	postComments, err := h.storage.PostComments.GetPostComments(post.Id, skip, limit) // get post comments by recency (top comment will be most recent)

	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalCommentsCount, err := h.storage.PostComments.GetPostCommentsCount(post.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalCommentsCount) / float64(limit)))

	type Response struct {
		Success   bool                              `json:"success"`
		Comments  []storage.PostCommentWithMetaData `json:"comments"`
		NoOfPages int                               `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, Comments: postComments, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}

}

// no auth required
func (h *Handler) GetCommentRepliesHandler(w http.ResponseWriter, r *http.Request) {

	commentId, err := strconv.Atoi(chi.URLParam(r, "commentId"))
	if err != nil {
		writeJSONError(w, "invalid request param commentId", http.StatusBadRequest)
		return
	}

	comment, err := h.storage.PostComments.GetPostCommentById(commentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post comment not found", http.StatusNotFound)
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

	commentReplies, err := h.storage.PostComments.GetCommentReplies(comment.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalCommentRepliesCount, err := h.storage.PostComments.GetCommentRepliesCount(comment.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalCommentRepliesCount) / float64(limit)))

	type Response struct {
		Success        bool                              `json:"success"`
		CommentReplies []storage.PostCommentWithMetaData `json:"comment_replies"`
		NoOfPages      int                               `json:"no_of_pages"`
	}

	if err := writeJSON(w, Response{Success: true, CommentReplies: commentReplies, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

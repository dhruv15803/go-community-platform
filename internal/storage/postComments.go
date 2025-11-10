package storage

import "github.com/jmoiron/sqlx"

type PostComment struct {
	Id               int     `db:"id" json:"id"`
	CommentContent   string  `db:"comment_content" json:"comment_content"`
	CommentOwnerId   int     `db:"comment_owner_id" json:"comment_owner_id"`
	PostId           int     `db:"post_id" json:"post_id"`
	ParentCommentId  *int    `db:"parent_comment_id" json:"parent_comment_id"`
	CommentCreatedAt string  `db:"comment_created_at" json:"comment_created_at"`
	CommentUpdatedAt *string `db:"comment_updated_at" json:"comment_updated_at"`
}

type PostCommentLike struct {
	LikedById          int    `db:"liked_by_id" json:"liked_by_id"`
	LikedPostCommentId int    `db:"liked_post_comment_id" json:"liked_post_comment_id"`
	LikedAt            string `db:"liked_at" json:"liked_at"`
}

type PostCommentWithUser struct {
	PostComment
	CommentOwner User `json:"comment_owner"`
}

type PostCommentWithMetaData struct {
	PostCommentWithUser
	CommentLikesCount int `json:"comment_likes_count"`
}

type PostCommentRepo struct {
	db *sqlx.DB
}

func NewPostCommentRepo(db *sqlx.DB) *PostCommentRepo {
	return &PostCommentRepo{
		db: db,
	}
}

func (c *PostCommentRepo) CreatePostComment(commentContent string, commentOwnerId int, postId int) (*PostComment, error) {

	var postComment PostComment

	query := `INSERT INTO post_comments(comment_content,comment_owner_id,post_id) VALUES($1,$2,$3) RETURNING 
	id,comment_content,comment_owner_id,post_id,parent_comment_id,comment_created_at,comment_updated_at`

	if err := c.db.QueryRowx(query, commentContent, commentOwnerId, postId).StructScan(&postComment); err != nil {
		return nil, err
	}

	return &postComment, nil
}

func (c *PostCommentRepo) CreateChildPostComment(commentContent string, commentOwnerId int, postId int, parentCommentId int) (*PostComment, error) {

	var postComment PostComment

	query := `INSERT INTO post_comments(comment_content,comment_owner_id,post_id,parent_comment_id) VALUES($1,$2,$3,$4) RETURNING 
	id,comment_content,comment_owner_id,post_id,parent_comment_id,comment_created_at,comment_updated_at`

	if err := c.db.QueryRowx(query, commentContent, commentOwnerId, postId, parentCommentId).StructScan(&postComment); err != nil {
		return nil, err
	}

	return &postComment, nil
}

func (c *PostCommentRepo) GetPostCommentById(id int) (*PostComment, error) {

	var postComment PostComment

	query := `SELECT id, comment_content, comment_owner_id, post_id, parent_comment_id, comment_created_at, comment_updated_at 
	FROM post_comments WHERE id=$1`

	if err := c.db.QueryRowx(query, id).StructScan(&postComment); err != nil {
		return nil, err
	}

	return &postComment, nil
}

func (c *PostCommentRepo) DeletePostCommentById(id int) error {

	query := `DELETE FROM post_comments WHERE id=$1`

	_, err := c.db.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

func (c *PostCommentRepo) CheckCommentLike(userId int, commentId int) (bool, error) {

	var commentLike PostCommentLike

	query := `SELECT liked_by_id, liked_post_comment_id, liked_at 
	FROM post_comment_likes WHERE liked_by_id=$1 AND liked_post_comment_id=$2`

	if err := c.db.QueryRowx(query, userId, commentId).StructScan(&commentLike); err != nil {
		return false, err
	}

	return true, nil

}

func (c *PostCommentRepo) CreateCommentLike(userId int, commentId int) (*PostCommentLike, error) {

	var commentLike PostCommentLike

	query := `INSERT INTO post_comment_likes(liked_by_id, liked_post_comment_id) VALUES($1,$2) RETURNING liked_by_id,liked_post_comment_id,liked_at`

	if err := c.db.QueryRowx(query, userId, commentId).StructScan(&commentLike); err != nil {
		return nil, err
	}

	return &commentLike, nil

}

func (c *PostCommentRepo) RemoveCommentLike(userId int, commentId int) error {

	query := `DELETE FROM post_comment_likes WHERE liked_by_id=$1 AND liked_post_comment_id=$2`

	_, err := c.db.Exec(query, userId, commentId)
	if err != nil {
		return err
	}

	return nil

}

func (c *PostCommentRepo) GetPostComments(postId int, offset int, limit int) ([]PostCommentWithMetaData, error) {

	var postComments []PostCommentWithMetaData

	query := `
	SELECT
	  	pc.id, comment_content, comment_owner_id, post_id, parent_comment_id, comment_created_at, comment_updated_at,
	  	u.id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, 
       	verified_at, created_at, updated_at,COUNT(DISTINCT(cl.liked_by_id)) AS comment_likes_count
	FROM
	  post_comments AS pc
	  INNER JOIN users AS u ON pc.comment_owner_id = u.id
	  LEFT JOIN post_comment_likes AS cl ON pc.id = cl.liked_post_comment_id
	WHERE
	  post_id = $1 AND parent_comment_id IS NULL
	GROUP BY 
	  pc.id, u.id
	ORDER BY 
	  comment_created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := c.db.Queryx(query, postId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var postComment PostCommentWithMetaData

		if err := rows.Scan(&postComment.Id, &postComment.CommentContent, &postComment.CommentOwnerId, &postComment.PostId,
			&postComment.ParentCommentId, &postComment.CommentCreatedAt, &postComment.CommentUpdatedAt, &postComment.CommentOwner.Id,
			&postComment.CommentOwner.Email, &postComment.CommentOwner.Password, &postComment.CommentOwner.Username,
			&postComment.CommentOwner.IsVerified, &postComment.CommentOwner.Role, &postComment.CommentOwner.UserImage, &postComment.CommentOwner.Bio,
			&postComment.CommentOwner.Location, &postComment.CommentOwner.DateOfBirth, &postComment.CommentOwner.VerifiedAt,
			&postComment.CommentOwner.CreatedAt, &postComment.CommentOwner.UpdatedAt, &postComment.CommentLikesCount); err != nil {
			return nil, err
		}

		postComments = append(postComments, postComment)
	}

	return postComments, nil

}

func (c *PostCommentRepo) GetPostCommentsCount(postId int) (int, error) {

	var totalPostCommentsCount int

	query := `SELECT COUNT(*) FROM post_comments WHERE post_id=$1 AND parent_comment_id IS NULL`

	if err := c.db.QueryRowx(query, postId).Scan(&totalPostCommentsCount); err != nil {
		return -1, err
	}

	return totalPostCommentsCount, nil

}

func (c *PostCommentRepo) GetCommentReplies(commentId int, offset int, limit int) ([]PostCommentWithMetaData, error) {

	var commentReplies []PostCommentWithMetaData

	query := `
	SELECT 
	    pc.id, comment_content, comment_owner_id, post_id, parent_comment_id, comment_created_at, comment_updated_at,
		u.id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, 
		verified_at, created_at, updated_at,COUNT(DISTINCT(cl.liked_by_id)) AS comment_likes_count 
	FROM 
	    post_comments AS pc INNER JOIN users AS u ON pc.comment_owner_id=u.id 
	    LEFT JOIN post_comment_likes AS cl ON pc.id = cl.liked_post_comment_id	
	WHERE 
	    parent_comment_id=$1
	GROUP BY 
	    pc.id,u.id
	ORDER BY 
	    comment_created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := c.db.Queryx(query, commentId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var postComment PostCommentWithMetaData

		if err := rows.Scan(&postComment.Id, &postComment.CommentContent, &postComment.CommentOwnerId, &postComment.PostId,
			&postComment.ParentCommentId, &postComment.CommentCreatedAt, &postComment.CommentUpdatedAt, &postComment.CommentOwner.Id,
			&postComment.CommentOwner.Email, &postComment.CommentOwner.Password, &postComment.CommentOwner.Username,
			&postComment.CommentOwner.IsVerified, &postComment.CommentOwner.Role, &postComment.CommentOwner.UserImage, &postComment.CommentOwner.Bio,
			&postComment.CommentOwner.Location, &postComment.CommentOwner.DateOfBirth, &postComment.CommentOwner.VerifiedAt,
			&postComment.CommentOwner.CreatedAt, &postComment.CommentOwner.UpdatedAt, &postComment.CommentLikesCount); err != nil {
			return nil, err
		}

		commentReplies = append(commentReplies, postComment)

	}

	return commentReplies, nil

}

func (c *PostCommentRepo) GetCommentRepliesCount(commentId int) (int, error) {

	var totalCommentRepliesCount int

	query := `SELECT COUNT(*) FROM post_comments WHERE parent_comment_id=$1`

	if err := c.db.QueryRowx(query, commentId).Scan(&totalCommentRepliesCount); err != nil {
		return -1, err
	}

	return totalCommentRepliesCount, nil

}

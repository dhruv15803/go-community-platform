package storage

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Post struct {
	Id              int     `db:"id" json:"id"`
	PostTitle       string  `db:"post_title" json:"post_title"`
	PostContent     string  `db:"post_content" json:"post_content"`
	PostOwnerId     int     `db:"post_owner_id" json:"post_owner_id"`
	PostCommunityId int     `db:"post_community_id" json:"post_community_id"`
	PostCreatedAt   string  `db:"post_created_at" json:"post_created_at"`
	PostUpdatedAt   *string `db:"post_updated_at" json:"post_updated_at"`
}

type PostLike struct {
	LikedById   int    `db:"liked_by_id" json:"liked_by_id"`
	LikedPostId int    `db:"liked_post_id" json:"liked_post_id"`
	LikedAt     string `db:"liked_at" json:"liked_at"`
}

type PostBookmark struct {
	BookmarkedById   int    `db:"bookmarked_by_id" json:"bookmarked_by_id"`
	BookmarkedPostId int    `db:"bookmarked_post_id" json:"bookmarked_post_id"`
	BookmarkedAt     string `db:"bookmarked_at" json:"bookmarked_at"`
}

type PostImage struct {
	Id           int    `db:"id" json:"id"`
	PostImageUrl string `db:"post_image_url" json:"post_image_url"`
	PostId       int    `db:"post_id" json:"post_id"`
}

type PostWithImages struct {
	Post
	PostImages []PostImage `json:"post_images"`
}

type PostWithMetaData struct {
	PostWithImages
	PostOwner          User `json:"post_owner"`
	PostLikesCount     int  `json:"post_likes_count"`
	PostCommentsCount  int  `json:"post_comments_count"`
	PostBookmarksCount int  `json:"post_bookmarks_count"`
}

type PostRepo struct {
	db *sqlx.DB
}

func NewPostRepo(db *sqlx.DB) *PostRepo {
	return &PostRepo{
		db: db,
	}
}

func (p *PostRepo) CreatePostWithImages(postTitle string, postContent string, postOwnerId int, postCommunityId int, postImageUrls []string) (*PostWithImages, error) {

	var postWithImages PostWithImages
	var post Post
	var postImages []PostImage

	tx, err := p.db.Beginx()
	if err != nil {
		return nil, err
	}

	var rollBackErr error

	defer func() {
		if rollBackErr != nil {
			tx.Rollback()
		}
	}()

	createPostQuery := `INSERT INTO posts(post_title,post_content,post_owner_id,post_community_id) VALUES($1,$2,$3,$4) RETURNING 
    id,post_title,post_content,post_owner_id,post_community_id,post_created_at,post_updated_at`

	if err := tx.QueryRowx(createPostQuery, postTitle, postContent, postOwnerId, postCommunityId).StructScan(&post); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	for _, postImageUrl := range postImageUrls {

		var postImage PostImage

		createPostImageQuery := `INSERT INTO post_images(post_image_url,post_id) VALUES($1,$2) RETURNING id,post_image_url,post_id`

		if err := tx.QueryRowx(createPostImageQuery, postImageUrl, post.Id).StructScan(&postImage); err != nil {
			rollBackErr = err
			return nil, rollBackErr
		}

		postImages = append(postImages, postImage)

	}

	postWithImages.Post = post
	postWithImages.PostImages = postImages

	if err = tx.Commit(); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	return &postWithImages, nil
}

func (p *PostRepo) CreatePost(postTitle string, postContent string, postOwnerId int, postCommunityId int) (*Post, error) {

	var post Post

	query := `INSERT INTO posts(post_title,post_content,post_owner_id,post_community_id) VALUES($1,$2,$3,$4) RETURNING 
	id,post_title,post_content,post_owner_id,post_community_id,post_created_at,post_updated_at`

	if err := p.db.QueryRowx(query, postTitle, postContent, postOwnerId, postCommunityId).StructScan(&post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (p *PostRepo) GetPostById(id int) (*Post, error) {

	var post Post

	query := `SELECT id, post_title, post_content, post_owner_id, post_community_id, post_created_at, post_updated_at 
	FROM posts WHERE id=$1`

	if err := p.db.QueryRowx(query, id).StructScan(&post); err != nil {
		return nil, err
	}

	return &post, nil

}

func (p *PostRepo) DeletePostById(id int) error {

	query := `DELETE FROM posts WHERE id=$1`

	_, err := p.db.Exec(query, id)
	if err != nil {
		return err
	}

	return nil

}

func (p *PostRepo) CheckPostLike(userId int, postId int) (bool, error) {

	var postLike PostLike

	query := `SELECT liked_by_id,liked_post_id,liked_at 
	FROM post_likes WHERE liked_by_id=$1 AND liked_post_id=$2`

	if err := p.db.QueryRowx(query, userId, postId).StructScan(&postLike); err != nil {
		return false, err
	}

	return true, nil
}

func (p *PostRepo) CreatePostLike(userId int, postId int) (*PostLike, error) {

	var postLike PostLike

	query := `INSERT INTO post_likes(liked_by_id,liked_post_id) VALUES($1,$2) RETURNING liked_by_id,liked_post_id,liked_at`

	if err := p.db.QueryRowx(query, userId, postId).StructScan(&postLike); err != nil {
		return nil, err
	}

	return &postLike, nil

}

func (p *PostRepo) RemovePostLike(userId int, postId int) error {

	query := `DELETE FROM post_likes WHERE liked_by_id=$1 AND liked_post_id=$2`

	_, err := p.db.Exec(query, userId, postId)
	if err != nil {
		return err
	}

	return nil

}

func (p *PostRepo) CheckPostBookmark(userId int, postId int) (bool, error) {

	var postBookmark PostBookmark

	query := `SELECT bookmarked_by_id, bookmarked_post_id, bookmarked_at 
	FROM post_bookmarks WHERE bookmarked_by_id=$1 AND bookmarked_post_id=$2`

	if err := p.db.QueryRowx(query, userId, postId).StructScan(&postBookmark); err != nil {
		return false, err
	}

	return true, nil

}

func (p *PostRepo) CreatePostBookmark(userId int, postId int) (*PostBookmark, error) {

	var postBookmark PostBookmark

	query := `INSERT INTO post_bookmarks(bookmarked_by_id,bookmarked_post_id) VALUES($1,$2) RETURNING bookmarked_by_id,bookmarked_post_id,bookmarked_at`

	if err := p.db.QueryRowx(query, userId, postId).StructScan(&postBookmark); err != nil {
		return nil, err
	}

	return &postBookmark, nil

}

func (p *PostRepo) RemovePostBookmark(userId int, postId int) error {

	query := `DELETE FROM post_bookmarks WHERE bookmarked_by_id=$1 AND bookmarked_post_id=$2`

	_, err := p.db.Exec(query, userId, postId)
	if err != nil {
		return err
	}

	return nil

}

func (p *PostRepo) GetCommunityPosts(communityId int, skip int, limit int, sortBy SortByStr, search string) ([]PostWithMetaData, error) {

	// if search is empty -> fetch all posts without filtering
	// else filter

	var posts []PostWithMetaData
	var query string
	var args []interface{}

	limitParam := 2
	offsetParam := 3

	whereClause := `WHERE
      post_community_id = $1
      AND pc.parent_comment_id IS NULL `

	args = append(args, communityId)

	if search != "" {
		whereClause += ` AND post_title ILIKE $2`
		searchArg := "%" + search + "%"
		args = append(args, searchArg)

		limitParam = 3
		offsetParam = 4
	}

	args = append(args, limit)
	args = append(args, skip)

	baseQuery := `SELECT
      p.id,
      post_title,
      post_content,
      post_owner_id,
      post_community_id,
      post_created_at,
      post_updated_at,
      u.id,
      email,
      password,
      username,
      is_verified,
      role,
      user_image,
      bio,
      location,
      date_of_birth,
      verified_at,
      created_at,
      updated_at,
      COUNT(DISTINCT (pl.liked_by_id)) AS post_likes_count,
      COUNT(DISTINCT (pc.id)) AS post_comments_count,
      COUNT(DISTINCT (pb.bookmarked_by_id)) AS post_bookmarks_count,
      0.0 as activity_score
    FROM
      posts AS p
      INNER JOIN users AS u ON p.post_owner_id = u.id
      LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
      LEFT JOIN post_comments AS pc ON p.id = pc.post_id
      LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
    `

	if sortBy == SortByNewest {

		query = fmt.Sprintf(`%s %s GROUP BY p.id, u.id ORDER BY post_created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, whereClause, limitParam, offsetParam)

	} else if sortBy == SortByTop {

		query = fmt.Sprintf("%s %s GROUP BY p.id,u.id ORDER BY post_likes_count DESC LIMIT $%d OFFSET $%d", baseQuery, whereClause, limitParam, offsetParam)

	} else if sortBy == SortByRelevance {

		baseQuery := `SELECT
      p.id,
      post_title,
      post_content,
      post_owner_id,
      post_community_id,
      post_created_at,
      post_updated_at,
      u.id,
      email,
      password,
      username,
      is_verified,
      role,
      user_image,
      bio,
      location,
      date_of_birth,
      verified_at,
      created_at,
      updated_at,
      COUNT(DISTINCT (pl.liked_by_id)) AS post_likes_count,
      COUNT(DISTINCT (pc.id)) AS post_comments_count,
      COUNT(DISTINCT (pb.bookmarked_by_id)) AS post_bookmarks_count
    FROM
      posts AS p
      INNER JOIN users AS u ON p.post_owner_id = u.id
      LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
      LEFT JOIN post_comments AS pc ON p.id = pc.post_id
      LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
    `

		selectClause := `SELECT
  *,
  (
    (
      0.3 * post_likes_count + 0.5 * post_comments_count + 0.2 * post_bookmarks_count
    ) / POWER(
      (
        EXTRACT(
          EPOCH
          FROM
            (NOW() - post_created_at)
        ) / 60
      ),
      2
    )
  ) AS activity_score`

		query = fmt.Sprintf(`%s FROM (%s %s GROUP BY p.id,u.id) ORDER BY activity_score DESC LIMIT $%d OFFSET $%d`, selectClause, baseQuery, whereClause, limitParam, offsetParam)

	} else {
		return nil, errors.New("Invalid SortBy")
	}

	rows, err := p.db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData
		var activityScore float64
		var postImages []PostImage

		if err := rows.Scan(

			&postWithMetaData.Id, &postWithMetaData.PostTitle, &postWithMetaData.PostContent,
			&postWithMetaData.PostOwnerId, &postWithMetaData.PostCommunityId, &postWithMetaData.PostCreatedAt,
			&postWithMetaData.PostUpdatedAt, &postWithMetaData.PostOwner.Id, &postWithMetaData.PostOwner.Email,
			&postWithMetaData.PostOwner.Password, &postWithMetaData.PostOwner.Username, &postWithMetaData.PostOwner.IsVerified,
			&postWithMetaData.PostOwner.Role, &postWithMetaData.PostOwner.UserImage, &postWithMetaData.PostOwner.Bio,
			&postWithMetaData.PostOwner.Location, &postWithMetaData.PostOwner.DateOfBirth, &postWithMetaData.PostOwner.VerifiedAt,
			&postWithMetaData.PostOwner.CreatedAt, &postWithMetaData.PostOwner.UpdatedAt, &postWithMetaData.PostLikesCount,
			&postWithMetaData.PostCommentsCount, &postWithMetaData.PostBookmarksCount, &activityScore); err != nil {

			return nil, err

		}

		// a post has many images
		postId := postWithMetaData.Id

		postImagesQuery := `SELECT id, post_image_url, post_id 
		FROM post_images WHERE post_id=$1`

		imageRows, err := p.db.Queryx(postImagesQuery, postId)
		if err != nil {
			return nil, err
		}
		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return nil, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		posts = append(posts, postWithMetaData)
	}

	return posts, nil
}

func (p *PostRepo) GetCommunityPostsCount(communityId int, search string) (int, error) {

	var count int
	var args []interface{}

	args = append(args, communityId)

	whereClause := `WHERE post_community_id=$1`

	if search != "" {
		whereClause += ` AND post_title ILIKE $2`
		searchArg := "%" + search + "%"
		args = append(args, searchArg)
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM posts %s`, whereClause)

	err := p.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}

// get posts from top N user communities with most members
func (p *PostRepo) GetUserPostsFeed(userId int, n int, skip int, limit int, sortBy SortByStr) ([]PostWithMetaData, error) {

	var posts []PostWithMetaData
	var query string

	baseQuery := `
      SELECT p.id,p.post_title,p.post_content,p.post_owner_id,
  p.post_community_id,p.post_created_at,p.post_updated_at,
  u.id,u.email,u.password,u.username,u.is_verified,u.role,
  u.user_image,u.bio,u.location,u.date_of_birth,u.verified_at,
  u.created_at,u.updated_at,
  COUNT(DISTINCT(pl.liked_by_id)) AS post_likes_count,
  COUNT(DISTINCT(pc.id)) AS post_comments_count,
  COUNT(DISTINCT(pb.bookmarked_by_id)) AS post_bookmarks_count, 
  0.0 AS activity_score
      FROM posts 
AS p INNER JOIN users AS u ON p.post_owner_id=u.id
LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
LEFT JOIN post_comments AS pc ON p.id = pc.post_id
LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
  WHERE post_community_id
IN (
  SELECT community_id
FROM (
  SELECT ucl.community_id, COUNT(DISTINCT(ucr.user_id)) AS members_count FROM user_communities AS ucl
INNER JOIN user_communities AS ucr ON ucl.community_id = ucr.community_id
WHERE ucl.user_id=$1
GROUP BY ucl.user_id,ucl.community_id,ucl.joined_at
ORDER BY members_count DESC
LIMIT $2 OFFSET 0
  )
)
AND pc.parent_comment_id IS NULL
GROUP BY p.id,u.id`

	if sortBy == SortByTop {

		query = fmt.Sprintf("%s\nORDER BY post_likes_count DESC\nLIMIT $3 OFFSET $4", baseQuery)

	} else if sortBy == SortByNewest {

		query = fmt.Sprintf("%s\nORDER BY p.post_created_at DESC\nLIMIT $3 OFFSET $4", baseQuery)

	} else if sortBy == SortByRelevance {

		baseQuery = `SELECT p.id,p.post_title,p.post_content,p.post_owner_id,
  p.post_community_id,p.post_created_at,p.post_updated_at,
  u.id,u.email,u.password,u.username,u.is_verified,u.role,
  u.user_image,u.bio,u.location,u.date_of_birth,u.verified_at,
  u.created_at,u.updated_at,
  COUNT(DISTINCT(pl.liked_by_id)) AS post_likes_count,
  COUNT(DISTINCT(pc.id)) AS post_comments_count,
  COUNT(DISTINCT(pb.bookmarked_by_id)) AS post_bookmarks_count FROM posts 
AS p INNER JOIN users AS u ON p.post_owner_id=u.id
LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
LEFT JOIN post_comments AS pc ON p.id = pc.post_id
LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
  WHERE post_community_id
IN (
  SELECT community_id
FROM (
  SELECT ucl.community_id, COUNT(DISTINCT(ucr.user_id)) AS members_count FROM user_communities AS ucl
INNER JOIN user_communities AS ucr ON ucl.community_id = ucr.community_id
WHERE ucl.user_id=$1
GROUP BY ucl.user_id,ucl.community_id,ucl.joined_at
ORDER BY members_count DESC
LIMIT $2 OFFSET 0
  )
)
AND pc.parent_comment_id IS NULL
GROUP BY p.id,u.id`

		selectClause := `SELECT *,(
    (
      0.3 * post_likes_count + 0.5 * post_comments_count + 0.2 * post_bookmarks_count
    ) / POWER(
      (
        EXTRACT(
          EPOCH
          FROM
            (NOW() - post_created_at)
        ) / 60
      ),
      2
    )
  ) AS activity_score`

		query = fmt.Sprintf("%s\n FROM (%s) ORDER BY activity_score DESC\n LIMIT $3 OFFSET $4", selectClause, baseQuery)

	} else {
		return nil, errors.New("invalid sortBy")
	}

	rows, err := p.db.Queryx(query, userId, n, limit, skip) // n is the top n communities being selected according to members count
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData
		var activityScore float64
		var postImages []PostImage

		if err := rows.Scan(
			&postWithMetaData.Id, &postWithMetaData.PostTitle, &postWithMetaData.PostContent,
			&postWithMetaData.PostOwnerId, &postWithMetaData.PostCommunityId, &postWithMetaData.PostCreatedAt,
			&postWithMetaData.PostUpdatedAt, &postWithMetaData.PostOwner.Id, &postWithMetaData.PostOwner.Email,
			&postWithMetaData.PostOwner.Password, &postWithMetaData.PostOwner.Username, &postWithMetaData.PostOwner.IsVerified,
			&postWithMetaData.PostOwner.Role, &postWithMetaData.PostOwner.UserImage, &postWithMetaData.PostOwner.Bio,
			&postWithMetaData.PostOwner.Location, &postWithMetaData.PostOwner.DateOfBirth, &postWithMetaData.PostOwner.VerifiedAt,
			&postWithMetaData.PostOwner.CreatedAt, &postWithMetaData.PostOwner.UpdatedAt, &postWithMetaData.PostLikesCount,
			&postWithMetaData.PostCommentsCount, &postWithMetaData.PostBookmarksCount, &activityScore); err != nil {
			return nil, err
		}

		imagesQuery := `SELECT id, post_image_url, post_id 
		FROM post_images WHERE post_id=$1`

		imageRows, err := p.db.Queryx(imagesQuery, postWithMetaData.Id)
		if err != nil {
			return nil, err
		}
		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return nil, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		posts = append(posts, postWithMetaData)
	}

	return posts, nil
}

func (p *PostRepo) GetUserPostsFeedCount(userId int, n int) (int, error) {

	var totalCount int

	query := `SELECT COUNT(*) FROM posts WHERE post_community_id IN (
		SELECT community_id
	FROM (
	  SELECT ucl.community_id, COUNT(DISTINCT(ucr.user_id)) AS members_count FROM user_communities AS ucl
	INNER JOIN user_communities AS ucr ON ucl.community_id = ucr.community_id
	WHERE ucl.user_id=$1
	GROUP BY ucl.user_id,ucl.community_id,ucl.joined_at
	ORDER BY members_count DESC
	LIMIT $2 OFFSET 0
))`

	if err := p.db.QueryRow(query, userId, n).Scan(&totalCount); err != nil {
		return -1, err
	}

	return totalCount, nil
}

func (p *PostRepo) GetPostsFeed(n int, skip int, limit int, sortBy SortByStr) ([]PostWithMetaData, error) {

	var posts []PostWithMetaData
	var query string

	baseQuery := `
      SELECT p.id,p.post_title,p.post_content,p.post_owner_id,
  p.post_community_id,p.post_created_at,p.post_updated_at,
  u.id,u.email,u.password,u.username,u.is_verified,u.role,
  u.user_image,u.bio,u.location,u.date_of_birth,u.verified_at,
  u.created_at,u.updated_at,
  COUNT(DISTINCT(pl.liked_by_id)) AS post_likes_count,
  COUNT(DISTINCT(pc.id)) AS post_comments_count,
  COUNT(DISTINCT(pb.bookmarked_by_id)) AS post_bookmarks_count, 
  0.0 AS activity_score
      FROM posts 
AS p INNER JOIN users AS u ON p.post_owner_id=u.id
LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
LEFT JOIN post_comments AS pc ON p.id = pc.post_id
LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
  WHERE post_community_id IN (
  SELECT community_id
  FROM (
    SELECT uc.community_id, COUNT(DISTINCT(uc.user_id)) AS members_count 
    FROM user_communities AS uc
    GROUP BY uc.community_id
    ORDER BY members_count DESC
    LIMIT $1 OFFSET 0
  )
)
AND pc.parent_comment_id IS NULL
GROUP BY p.id,u.id`

	if sortBy == SortByTop {

		query = fmt.Sprintf("%s\nORDER BY post_likes_count DESC\nLIMIT $2 OFFSET $3", baseQuery)

	} else if sortBy == SortByNewest {

		query = fmt.Sprintf("%s\nORDER BY p.post_created_at DESC\nLIMIT $2 OFFSET $3", baseQuery)

	} else if sortBy == SortByRelevance {

		baseQuery = `SELECT p.id,p.post_title,p.post_content,p.post_owner_id,
  p.post_community_id,p.post_created_at,p.post_updated_at,
  u.id,u.email,u.password,u.username,u.is_verified,u.role,
  u.user_image,u.bio,u.location,u.date_of_birth,u.verified_at,
  u.created_at,u.updated_at,
  COUNT(DISTINCT(pl.liked_by_id)) AS post_likes_count,
  COUNT(DISTINCT(pc.id)) AS post_comments_count,
  COUNT(DISTINCT(pb.bookmarked_by_id)) AS post_bookmarks_count FROM posts 
AS p INNER JOIN users AS u ON p.post_owner_id=u.id
LEFT JOIN post_likes AS pl ON p.id = pl.liked_post_id
LEFT JOIN post_comments AS pc ON p.id = pc.post_id
LEFT JOIN post_bookmarks AS pb ON p.id = pb.bookmarked_post_id
  WHERE post_community_id IN (
  SELECT community_id
  FROM (
    SELECT uc.community_id, COUNT(DISTINCT(uc.user_id)) AS members_count 
    FROM user_communities AS uc
    GROUP BY uc.community_id
    ORDER BY members_count DESC
    LIMIT $1 OFFSET 0
  )
)
AND pc.parent_comment_id IS NULL
GROUP BY p.id,u.id`

		selectClause := `SELECT *,(
    (
      0.3 * post_likes_count + 0.5 * post_comments_count + 0.2 * post_bookmarks_count
    ) / POWER(
      (
        EXTRACT(
          EPOCH
          FROM
            (NOW() - post_created_at)
        ) / 60
      ),
      2
    )
  ) AS activity_score `

		query = fmt.Sprintf("%s\n FROM (%s) ORDER BY activity_score DESC\n LIMIT $2 OFFSET $3", selectClause, baseQuery)

	} else {
		return nil, errors.New("invalid sortBy")
	}

	rows, err := p.db.Queryx(query, n, limit, skip) // n is the top n communities being selected according to members count
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData
		var activityScore float64
		var postImages []PostImage

		if err := rows.Scan(
			&postWithMetaData.Id, &postWithMetaData.PostTitle, &postWithMetaData.PostContent,
			&postWithMetaData.PostOwnerId, &postWithMetaData.PostCommunityId, &postWithMetaData.PostCreatedAt,
			&postWithMetaData.PostUpdatedAt, &postWithMetaData.PostOwner.Id, &postWithMetaData.PostOwner.Email,
			&postWithMetaData.PostOwner.Password, &postWithMetaData.PostOwner.Username, &postWithMetaData.PostOwner.IsVerified,
			&postWithMetaData.PostOwner.Role, &postWithMetaData.PostOwner.UserImage, &postWithMetaData.PostOwner.Bio,
			&postWithMetaData.PostOwner.Location, &postWithMetaData.PostOwner.DateOfBirth, &postWithMetaData.PostOwner.VerifiedAt,
			&postWithMetaData.PostOwner.CreatedAt, &postWithMetaData.PostOwner.UpdatedAt, &postWithMetaData.PostLikesCount,
			&postWithMetaData.PostCommentsCount, &postWithMetaData.PostBookmarksCount, &activityScore); err != nil {
			return nil, err
		}

		imagesQuery := `SELECT id, post_image_url, post_id 
		FROM post_images WHERE post_id=$1`

		imageRows, err := p.db.Queryx(imagesQuery, postWithMetaData.Id)
		if err != nil {
			return nil, err
		}
		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage
			if err := imageRows.StructScan(&postImage); err != nil {
				return nil, err
			}
			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		posts = append(posts, postWithMetaData)
	}

	return posts, nil
}

// GetExplorePostsFeedCount gets total count of posts from top N communities by member count
func (p *PostRepo) GetPostsFeedCount(n int) (int, error) {

	var totalCount int

	query := `SELECT COUNT(*) FROM posts WHERE post_community_id IN (
		SELECT community_id
		FROM (
			SELECT uc.community_id, COUNT(DISTINCT(uc.user_id)) AS members_count 
			FROM user_communities AS uc
			GROUP BY uc.community_id
			ORDER BY members_count DESC
			LIMIT $1 OFFSET 0
		)
	)`

	if err := p.db.QueryRow(query, n).Scan(&totalCount); err != nil {
		return -1, err
	}

	return totalCount, nil
}

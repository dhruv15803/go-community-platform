package storage

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type SortByStr string

const (
	SortByNewest    SortByStr = "new"
	SortByTop       SortByStr = "top" // sort posts by top post likes
	SortByRelevance SortByStr = "hot" // posts that are 'hot'
)

type Storage struct {
	Users                UserRepository
	Topics               TopicRepository
	UserTopicPreferences TopicPreferenceRepository
	TopicQueries         TopicQueryRepository
	Communities          CommunityRepository
	Posts                PostRepository
	PostComments         PostCommentRepository
}

func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{
		Users:                NewUserRepo(db),
		Topics:               NewTopicRepo(db),
		UserTopicPreferences: NewTopicPreferenceRepo(db),
		TopicQueries:         NewTopicQueryRepo(db),
		Communities:          NewCommunityRepo(db),
		Posts:                NewPostRepo(db),
		PostComments:         NewPostCommentRepo(db),
	}
}

type UserRepository interface {
	DeleteUserById(id int) error
	GetVerifiedUserByEmail(email string) (*User, error)
	CreateUserAndInvitation(email string, hashedPassword string, hashedToken string, expiration time.Time) (*User, error)
	CreateAdminUser(email string, hashedPassword string) (*User, error)
	CreateVerifiedUser(email string, hashedPassword string) (*User, error)
	ActivateUser(hashedToken string) (*User, error)
	GetUserById(id int) (*User, error)
}

type TopicRepository interface {
	GetTopicByTopicName(topicName string) (*Topic, error)
	CreateTopic(topicName string) (*Topic, error)
	GetTopicById(topicId int) (*Topic, error)
	DeleteTopicById(topicId int) error
	UpdateTopicById(topicId int, topicName string) (*Topic, error)
	GetTopics(offset int, limit int) ([]Topic, error)
	GetTopicsCount() (int, error)
}

type TopicPreferenceRepository interface {
	CreateUserTopicPreferences(userId int, topicIds []int) ([]UserTopicPreference, error)
	GetUserTopicPreference(userId int, topicId int) (*UserTopicPreference, error)
	DeleteUserTopicPreference(userId int, topicId int) error
	GetUserTopicPreferences(userId int) ([]UserTopicPreference, error)
}

type TopicQueryRepository interface {
	GetTopicsPrefferedByUser(userId int) ([]Topic, error)
}

type CommunityRepository interface {
	GetCommunityByName(communityName string) (*Community, error)
	CreateCommunityWithTopics(communityName string, communityDescription string, communityImage string, communityOwnerId int, communityTopicIds []int) (*CommunityWithTopics, error)
	GetCommunityById(communityId int) (*Community, error)
	CheckCommunityForUser(userId int, communityId int) (bool, error)
	JoinCommunity(userId int, communityId int) (*UserCommunity, error)
	LeaveCommunity(userId int, communityId int) error
	GetCommunityMembers(communityId int, offset int, limit int) ([]User, error)
	GetTotalCommunityMembersCount(communityId int) (int, error)
	GetCommunityProfile(communityId int) (*CommunityWithMetaData, error)
	GetRecommendedCommunitiesForUser(userId int, offset int, limit int) ([]CommunityWithMetaData, error)
	GetRecommendCommunitiesForUserCount(userId int) (int, error)
}

type PostRepository interface {
	CreatePost(postTitle string, postContent string, postOwnerId int, postCommunityId int) (*Post, error)
	CreatePostWithImages(postTitle string, postContent string, postOwnerId int, postCommunityId int, postImageUrls []string) (*PostWithImages, error)
	GetPostById(id int) (*Post, error)
	DeletePostById(id int) error
	CheckPostLike(userId int, postId int) (bool, error)
	CreatePostLike(userId int, postId int) (*PostLike, error)
	RemovePostLike(userId int, postId int) error
	CheckPostBookmark(userId int, postId int) (bool, error)
	CreatePostBookmark(userId int, postId int) (*PostBookmark, error)
	RemovePostBookmark(userId int, postId int) error
	GetCommunityPosts(communityId int, skip int, limit int, sortBy SortByStr, search string) ([]PostWithMetaData, error)
	GetCommunityPostsCount(communityId int, search string) (int, error)
}

type PostCommentRepository interface {
	DeletePostCommentById(id int) error
	GetPostCommentById(id int) (*PostComment, error)
	CreatePostComment(commentContent string, commentOwnerId int, postId int) (*PostComment, error)
	CreateChildPostComment(commentContent string, commentOwnerId int, postId int, parentCommentId int) (*PostComment, error)
	CheckCommentLike(userId int, commentId int) (bool, error)
	CreateCommentLike(userId int, commentId int) (*PostCommentLike, error)
	RemoveCommentLike(userId int, commentId int) error
	GetPostComments(postId int, offset int, limit int) ([]PostCommentWithMetaData, error)
	GetPostCommentsCount(postId int) (int, error)
	GetCommentReplies(commentId int, offset int, limit int) ([]PostCommentWithMetaData, error)
	GetCommentRepliesCount(commentId int) (int, error)
}

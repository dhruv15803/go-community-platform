package storage

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Storage struct {
	Users                UserRepository
	Topics               TopicRepository
	UserTopicPreferences TopicPreferenceRepository
	TopicQueryRepository TopicQueryRepository
}

func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{
		Users:                NewUserRepo(db),
		Topics:               NewTopicRepo(db),
		UserTopicPreferences: NewTopicPreferenceRepo(db),
		TopicQueryRepository: NewTopicQueryRepo(db),
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

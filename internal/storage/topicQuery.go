package storage

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type TopicQueryRepo struct {
	db *sqlx.DB
}

func NewTopicQueryRepo(db *sqlx.DB) *TopicQueryRepo {
	return &TopicQueryRepo{db: db}
}

func (q *TopicQueryRepo) GetTopicsPrefferedByUser(userId int) ([]Topic, error) {

	var userPrefferedTopics []Topic
	var userPrefferedTopicIds []int

	query := `SELECT user_id,topic_id 
	FROM user_topic_preferences WHERE user_id=$1`

	rows, err := q.db.Queryx(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var userTopicPreference UserTopicPreference

		if err := rows.StructScan(&userTopicPreference); err != nil {
			return nil, err
		}

		userPrefferedTopicIds = append(userPrefferedTopicIds, userTopicPreference.TopicId)
	}

	topicsQuery := `SELECT id,topic_name 
	FROM topics WHERE id = ANY($1)`

	topicRows, err := q.db.Queryx(topicsQuery, pq.Array(userPrefferedTopicIds))
	if err != nil {
		return nil, err
	}
	defer topicRows.Close()

	for topicRows.Next() {
		var topic Topic

		if err := topicRows.StructScan(&topic); err != nil {
			return nil, err
		}

		userPrefferedTopics = append(userPrefferedTopics, topic)
	}

	return userPrefferedTopics, nil
}

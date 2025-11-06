package storage

import "github.com/jmoiron/sqlx"

type Topic struct {
	Id        int    `db:"id" json:"id"`
	TopicName string `db:"topic_name" json:"topic_name"`
}

type TopicRepo struct {
	db *sqlx.DB
}

func NewTopicRepo(db *sqlx.DB) *TopicRepo {
	return &TopicRepo{db: db}
}

func (t *TopicRepo) GetTopicByTopicName(topicName string) (*Topic, error) {

	var topic Topic

	query := `SELECT id, topic_name 
	FROM topics WHERE topic_name=$1`

	if err := t.db.QueryRowx(query, topicName).StructScan(&topic); err != nil {
		return nil, err
	}

	return &topic, nil
}

func (t *TopicRepo) CreateTopic(topicName string) (*Topic, error) {

	var topic Topic

	query := `INSERT INTO topics(topic_name) VALUES($1) RETURNING 
	id,topic_name`

	if err := t.db.QueryRowx(query, topicName).StructScan(&topic); err != nil {
		return nil, err
	}

	return &topic, nil
}

func (t *TopicRepo) GetTopicById(topicId int) (*Topic, error) {

	var topic Topic

	query := `SELECT id, topic_name FROM topics WHERE id=$1`

	if err := t.db.QueryRowx(query, topicId).StructScan(&topic); err != nil {
		return nil, err
	}

	return &topic, nil
}

func (t *TopicRepo) DeleteTopicById(topicId int) error {

	query := `DELETE FROM topics WHERE id=$1`

	_, err := t.db.Exec(query, topicId)
	if err != nil {
		return err
	}

	return nil
}

func (t *TopicRepo) UpdateTopicById(topicId int, topicName string) (*Topic, error) {

	var topic Topic

	query := `UPDATE topics SET topic_name=$1 WHERE id=$2 RETURNING id,topic_name`

	if err := t.db.QueryRowx(query, topicName, topicId).StructScan(&topic); err != nil {
		return nil, err
	}

	return &topic, nil

}

func (t *TopicRepo) GetTopics(offset int, limit int) ([]Topic, error) {

	var topics []Topic

	query := `SELECT id, topic_name 
	FROM topics ORDER BY topic_name ASC LIMIT $1 OFFSET $2`

	rows, err := t.db.Queryx(query, limit, offset)
	if err != nil {
		return []Topic{}, err
	}
	defer rows.Close()

	for rows.Next() {

		var topic Topic

		if err := rows.StructScan(&topic); err != nil {
			return []Topic{}, err
		}

		topics = append(topics, topic)
	}

	return topics, nil

}

func (t *TopicRepo) GetTopicsCount() (int, error) {

	var totalTopicsCount int

	query := `SELECT COUNT(*) FROM topics`

	if err := t.db.QueryRowx(query).Scan(&totalTopicsCount); err != nil {
		return -1, err
	}

	return totalTopicsCount, nil
}

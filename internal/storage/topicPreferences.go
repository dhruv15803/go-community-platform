package storage

import "github.com/jmoiron/sqlx"

type UserTopicPreference struct {
	UserId  int `db:"user_id" json:"user_id"`
	TopicId int `db:"topic_id" json:"topic_id"`
}

type TopicPreferenceRepo struct {
	db *sqlx.DB
}

func NewTopicPreferenceRepo(db *sqlx.DB) *TopicPreferenceRepo {
	return &TopicPreferenceRepo{db: db}
}

func (p *TopicPreferenceRepo) CreateUserTopicPreferences(userId int, topicIds []int) ([]UserTopicPreference, error) {

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

	var userTopicPreferences []UserTopicPreference

	for _, topicId := range topicIds {

		var userTopicPreference UserTopicPreference
		query := `INSERT INTO user_topic_preferences(user_id,topic_id) VALUES($1,$2) RETURNING user_id,topic_id`

		if err := tx.QueryRowx(query, userId, topicId).StructScan(&userTopicPreference); err != nil {
			rollBackErr = err
			return nil, rollBackErr
		}

		userTopicPreferences = append(userTopicPreferences, userTopicPreference)
	}

	if err = tx.Commit(); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	return userTopicPreferences, nil
}

func (p *TopicPreferenceRepo) GetUserTopicPreference(userId int, topicId int) (*UserTopicPreference, error) {

	var topicPreference UserTopicPreference

	query := `SELECT user_id,topic_id 
	FROM user_topic_preferences WHERE user_id = $1 AND topic_id=$2`

	if err := p.db.QueryRowx(query, userId, topicId).StructScan(&topicPreference); err != nil {
		return nil, err
	}

	return &topicPreference, nil
}

func (p *TopicPreferenceRepo) DeleteUserTopicPreference(userId int, topicId int) error {

	query := `DELETE FROM user_topic_preferences WHERE user_id=$1 AND topic_id=$2`

	_, err := p.db.Exec(query, userId, topicId)
	if err != nil {
		return err
	}

	return nil
}

func (p *TopicPreferenceRepo) GetUserTopicPreferences(userId int) ([]UserTopicPreference, error) {

	var topicPreferences []UserTopicPreference

	query := `SELECT user_id,topic_id FROM user_topic_preferences WHERE user_id=$1`

	rows, err := p.db.Queryx(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var userTopicPreference UserTopicPreference

		if err := rows.StructScan(&userTopicPreference); err != nil {
			return nil, err
		}

		topicPreferences = append(topicPreferences, userTopicPreference)
	}

	return topicPreferences, nil

}

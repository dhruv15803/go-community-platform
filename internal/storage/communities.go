package storage

import "github.com/jmoiron/sqlx"

type Community struct {
	Id                   int     `db:"id" json:"id"`
	CommunityName        string  `db:"community_name" json:"community_name"`
	CommunityDescription *string `db:"community_description" json:"community_description"`
	CommunityImage       *string `db:"community_image" json:"community_image"`
	CommunityOwnerId     int     `db:"community_owner_id" json:"community_owner_id"`
	CommunityCreatedAt   string  `db:"community_created_at" json:"community_created_at"`
	CommunityUpdatedAt   *string `db:"community_updated_at" json:"community_updated_at"`
}

type CommunityTopic struct {
	CommunityId int `db:"community_id" json:"community_id"`
	TopicId     int `db:"topic_id" json:"topic_id"`
}

type UserCommunity struct {
	UserId      int    `db:"user_id" json:"user_id"`
	CommunityId int    `db:"community_id" json:"community_id"`
	JoinedAt    string `db:"joined_at" json:"joined_at"`
}

type CommunityWithTopics struct {
	Community
	CommunityTopics []Topic `json:"community_topics"`
}

type CommunityRepo struct {
	db *sqlx.DB
}

func NewCommunityRepo(db *sqlx.DB) *CommunityRepo {
	return &CommunityRepo{db: db}
}

func (c *CommunityRepo) GetCommunityByName(communityName string) (*Community, error) {

	var community Community

	query := `SELECT id, community_name, community_description, community_image, community_owner_id, community_created_at, community_updated_at 
	FROM communities WHERE community_name=$1`

	if err := c.db.QueryRowx(query, communityName).StructScan(&community); err != nil {
		return nil, err
	}

	return &community, nil
}

func (c *CommunityRepo) CreateCommunityWithTopics(communityName string, communityDescription string, communityImage string, communityOwnerId int, communityTopicIds []int) (*CommunityWithTopics, error) {

	var communityWithTopics CommunityWithTopics
	var community Community
	var topics []Topic // community topics

	tx, err := c.db.Beginx()
	if err != nil {
		return nil, err
	}
	var rollBackErr error

	defer func() {
		tx.Rollback()
	}()

	createCommunityQuery := `INSERT INTO communities(community_name,community_description,community_image,community_owner_id) VALUES($1,$2,$3,$4) RETURNING
	id, community_name, community_description, community_image, community_owner_id, community_created_at, community_updated_at`

	if err := tx.QueryRowx(createCommunityQuery, communityName, communityDescription, communityImage, communityOwnerId).StructScan(&community); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	createCommunityTopicQuery := `INSERT INTO community_topics(community_id,topic_id) VALUES($1,$2)`

	for _, topicId := range communityTopicIds {

		var topic Topic

		_, err := tx.Exec(createCommunityTopicQuery, community.Id, topicId)
		if err != nil {
			rollBackErr = err
			return nil, rollBackErr
		}

		topicQuery := `SELECT id,topic_name FROM topics WHERE id=$1`

		if err := tx.QueryRowx(topicQuery, topicId).StructScan(&topic); err != nil {
			rollBackErr = err
			return nil, rollBackErr
		}

		topics = append(topics, topic)
	}

	communityWithTopics.Community = community
	communityWithTopics.CommunityTopics = topics

	if err := tx.Commit(); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	return &communityWithTopics, nil
}

func (c *CommunityRepo) GetCommunityById(communityId int) (*Community, error) {

	var community Community

	query := `SELECT id, community_name, community_description, community_image, community_owner_id, community_created_at, community_updated_at 
	FROM communities WHERE id=$1`

	if err := c.db.QueryRowx(query, communityId).StructScan(&community); err != nil {
		return nil, err
	}

	return &community, nil
}

func (c *CommunityRepo) JoinCommunity(userId int, communityId int) (*UserCommunity, error) {

	var userCommunity UserCommunity

	query := `INSERT INTO user_communities(user_id,community_id) VALUES($1,$2) RETURNING user_id,community_id,joined_at`

	if err := c.db.QueryRowx(query, userId, communityId).StructScan(&userCommunity); err != nil {
		return nil, err
	}

	return &userCommunity, nil

}

func (c *CommunityRepo) LeaveCommunity(userId int, communityId int) error {

	query := `DELETE FROM user_communities WHERE user_id=$1 AND community_id=$2`

	_, err := c.db.Exec(query, userId, communityId)
	if err != nil {
		return err
	}

	return nil

}

func (c *CommunityRepo) CheckCommunityForUser(userId int, communityId int) (bool, error) {

	var userCommunity UserCommunity

	query := `SELECT * FROM user_communities WHERE user_id=$1 AND community_id=$2`

	if err := c.db.QueryRowx(query, userId, communityId).StructScan(&userCommunity); err != nil {
		return false, err
	}

	return true, nil
}

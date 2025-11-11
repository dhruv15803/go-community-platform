package storage

import (
	"github.com/jmoiron/sqlx"
)

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

type CommunityWithMetaData struct {
	CommunityWithTopics
	CommunityOwner        User `json:"community_owner"`
	CommunityMembersCount int  `json:"community_members_count"`
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

func (c *CommunityRepo) GetCommunityMembers(communityId int, offset int, limit int) ([]User, error) {

	var members []User

	query := `SELECT id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at 
	FROM users WHERE id IN (
		SELECT user_id FROM user_communities 
		WHERE community_id=$1 
		ORDER BY joined_at DESC
		LIMIT $2 OFFSET $3
	)`

	rows, err := c.db.Queryx(query, communityId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var member User

		if err := rows.StructScan(&member); err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	return members, nil
}

func (c *CommunityRepo) GetTotalCommunityMembersCount(communityId int) (int, error) {

	var totalMembersCount int

	query := `SELECT COUNT(user_id) FROM user_communities WHERE community_id=$1`

	if err := c.db.QueryRow(query, communityId).Scan(&totalMembersCount); err != nil {
		return -1, err
	}

	return totalMembersCount, nil

}

func (c *CommunityRepo) GetCommunityProfile(communityId int) (*CommunityWithMetaData, error) {

	var community CommunityWithMetaData
	var communityTopics []Topic
	var totalCommunityMembersCount int

	query := `SELECT c.id, community_name, community_description, community_image, community_owner_id, community_created_at, community_updated_at,
       u.id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at
	FROM communities AS c INNER JOIN users AS u ON c.community_owner_id=u.id WHERE c.id=$1`

	if err := c.db.QueryRowx(query, communityId).Scan(&community.Id, &community.CommunityName, &community.CommunityDescription,
		&community.CommunityImage, &community.CommunityOwnerId, &community.CommunityCreatedAt, &community.CommunityUpdatedAt,
		&community.CommunityOwner.Id, &community.CommunityOwner.Email, &community.CommunityOwner.Password, &community.CommunityOwner.Username,
		&community.CommunityOwner.IsVerified, &community.CommunityOwner.Role, &community.CommunityOwner.UserImage, &community.CommunityOwner.Bio,
		&community.CommunityOwner.Location, &community.CommunityOwner.DateOfBirth, &community.CommunityOwner.VerifiedAt, &community.CommunityOwner.CreatedAt,
		&community.CommunityOwner.UpdatedAt); err != nil {
		return nil, err
	}

	// community topics and members count

	topicsQuery := `SELECT id, topic_name 
	FROM topics WHERE id IN (SELECT topic_id FROM community_topics WHERE community_id=$1)`

	topicRows, err := c.db.Queryx(topicsQuery, communityId)
	if err != nil {
		return nil, err
	}
	defer topicRows.Close()

	for topicRows.Next() {

		var topic Topic

		if err := topicRows.StructScan(&topic); err != nil {
			return nil, err
		}

		communityTopics = append(communityTopics, topic)
	}

	communityMembersCountQuery := `SELECT COUNT(user_id) FROM user_communities WHERE community_id=$1`

	if err := c.db.QueryRowx(communityMembersCountQuery, communityId).Scan(&totalCommunityMembersCount); err != nil {
		return nil, err
	}

	community.CommunityTopics = communityTopics
	community.CommunityMembersCount = totalCommunityMembersCount

	return &community, nil
}

func (c *CommunityRepo) GetRecommendedCommunitiesForUser(userId int, offset int, limit int) ([]CommunityWithMetaData, error) {

	var communities []CommunityWithMetaData

	query := `SELECT c.id, community_name, community_description, community_image, community_owner_id, community_created_at, community_updated_at,
       u.id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at,
       COUNT(uc.user_id) AS members_count
	FROM communities AS c INNER JOIN users AS u ON c.community_owner_id=u.id   
    LEFT JOIN user_communities AS uc ON c.id = uc.community_id
  	WHERE c.id IN (
    SELECT DISTINCT(community_id) FROM community_topics WHERE topic_id IN (
    	SELECT topic_id FROM user_topic_preferences WHERE user_id=$1
	)) 
  GROUP BY c.id , u.id
  ORDER BY members_count DESC
  LIMIT $2 OFFSET $3`

	rows, err := c.db.Queryx(query, userId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var community CommunityWithMetaData

		if err := rows.Scan(&community.Id, &community.CommunityName, &community.CommunityDescription,
			&community.CommunityImage, &community.CommunityOwnerId, &community.CommunityCreatedAt,
			&community.CommunityUpdatedAt, &community.CommunityOwner.Id, &community.CommunityOwner.Email,
			&community.CommunityOwner.Password, &community.CommunityOwner.Username, &community.CommunityOwner.IsVerified,
			&community.CommunityOwner.Role, &community.CommunityOwner.UserImage, &community.CommunityOwner.Bio,
			&community.CommunityOwner.Location, &community.CommunityOwner.DateOfBirth, &community.CommunityOwner.VerifiedAt,
			&community.CommunityOwner.CreatedAt, &community.CommunityOwner.UpdatedAt, &community.CommunityMembersCount); err != nil {
			return nil, err
		}

		// get community topics

		var communityTopics []Topic

		topicsQuery := `SELECT id, topic_name FROM topics WHERE id IN (
    		SELECT topic_id FROM community_topics WHERE community_id=$1
		)`

		topicRows, err := c.db.Queryx(topicsQuery, community.Id)
		if err != nil {
			return nil, err
		}
		defer topicRows.Close()

		for topicRows.Next() {

			var topic Topic

			if err := topicRows.StructScan(&topic); err != nil {
				return nil, err
			}

			communityTopics = append(communityTopics, topic)
		}

		community.CommunityTopics = communityTopics

		communities = append(communities, community)
	}

	return communities, nil
}

func (c *CommunityRepo) GetRecommendCommunitiesForUserCount(userId int) (int, error) {

	var totalRecommendedCommunitiesCount int

	query := `SELECT COUNT(DISTINCT(community_id)) FROM community_topics WHERE topic_id IN (
    	SELECT topic_id FROM user_topic_preferences WHERE user_id=$1
	)`

	if err := c.db.QueryRowx(query, userId).Scan(&totalRecommendedCommunitiesCount); err != nil {
		return -1, err
	}

	return totalRecommendedCommunitiesCount, nil
}

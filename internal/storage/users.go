package storage

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

type User struct {
	Id          int     `db:"id" json:"id"`
	Email       string  `db:"email" json:"email"`
	Password    string  `db:"password" json:"-"`
	Username    *string `db:"username" json:"username"`
	IsVerified  bool    `db:"is_verified" json:"is_verified"`
	Role        string  `db:"role" json:"role"`
	UserImage   *string `db:"user_image" json:"user_image"`
	Bio         *string `db:"bio" json:"bio"`
	Location    *string `db:"location" json:"location"`
	DateOfBirth *string `db:"date_of_birth" json:"date_of_birth"`
	VerifiedAt  *string `db:"verified_at" json:"verified_at"`
	CreatedAt   string  `db:"created_at" json:"created_at"`
	UpdatedAt   *string `db:"updated_at" json:"updated_at"`
}

type UserInvitation struct {
	Token      string `db:"token" json:"token"`
	UserId     int    `db:"user_id" json:"user_id"`
	Expiration string `db:"expiration" json:"expiration"`
}

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func (u *UserRepo) DeleteUserById(id int) error {
	//TODO implement me
	return nil
}

func (u *UserRepo) GetVerifiedUserByEmail(email string) (*User, error) {

	var user User

	query := `SELECT id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at 
	FROM users WHERE email=$1 AND is_verified=TRUE`

	if err := u.db.QueryRowx(query, email).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) CreateUserAndInvitation(email string, hashedPassword string, hashedToken string, expiration time.Time) (*User, error) {

	var user User

	tx, err := u.db.Beginx()
	if err != nil {
		return nil, err
	}
	var rollBackErr error
	defer func() {
		if rollBackErr != nil {
			tx.Rollback()
		}
	}()

	createUserQuery := `INSERT INTO users(email,password) VALUES($1,$2) RETURNING 
	id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at`

	if err := tx.QueryRowx(createUserQuery, email, hashedPassword).StructScan(&user); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	createUserInvitationQuery := `INSERT INTO user_invitations(token, user_id, expiration) VALUES($1,$2,$3)`

	_, err = tx.Exec(createUserInvitationQuery, hashedToken, user.Id, expiration)
	if err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	if err = tx.Commit(); err != nil {
		rollBackErr = err
		return nil, rollBackErr
	}

	return &user, nil
}

func (u *UserRepo) ActivateUser(hashedToken string) (*User, error) {

	var user User
	var userInvitation UserInvitation

	query := `SELECT token, user_id, expiration 
	FROM user_invitations WHERE token=$1 AND expiration > $2`

	if err := u.db.QueryRowx(query, hashedToken, time.Now()).StructScan(&userInvitation); err != nil {
		return nil, err
	}

	userId := userInvitation.UserId

	updateUserQuery := `UPDATE users SET is_verified=TRUE,verified_at=$2 WHERE id=$1 RETURNING
	id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at`

	if err := u.db.QueryRowx(updateUserQuery, userId, time.Now()).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) GetUserById(id int) (*User, error) {

	var user User

	query := `SELECT id, email, password, username, is_verified, role, user_image, 
	bio, location, date_of_birth, verified_at, created_at, updated_at
	FROM users WHERE id=$1`

	if err := u.db.QueryRowx(query, id).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) CreateAdminUser(email string, hashedPassword string) (*User, error) {

	var user User

	query := `INSERT INTO users(email,password,is_verified,role,verified_at) VALUES($1,$2,$3,$4,$5) RETURNING 
	id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at`

	if err := u.db.QueryRowx(query, email, hashedPassword, true, UserRoleAdmin, time.Now()).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil

}

func (u *UserRepo) CreateVerifiedUser(email string, hashedPassword string) (*User, error) {

	var user User

	query := `INSERT INTO users(email,password,is_verified,verified_at) VALUES($1,$2,$3,$4) RETURNING 
	id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at`

	if err := u.db.QueryRowx(query, email, hashedPassword, true, time.Now()).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) GetUserByUsername(username string) (*User, error) {

	var user User

	query := `SELECT id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at 
	FROM users WHERE username=$1`

	if err := u.db.QueryRowx(query, username).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRepo) UpdateUsernameById(id int, username string) (*User, error) {

	var user User

	query := `UPDATE users SET username=$1 WHERE id=$2
	RETURNING id, email, password, username, is_verified, role, user_image, bio, location, date_of_birth, verified_at, created_at, updated_at`

	if err := u.db.QueryRowx(query, username, id).StructScan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

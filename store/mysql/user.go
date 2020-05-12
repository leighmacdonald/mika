package mysql

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

const ErrNoResults = "sql: no rows in result set"

// UserStore is the MySQL backed store.UserStore implementation
type UserStore struct {
	db      *sqlx.DB
	users   map[string]model.User
	usersMx sync.RWMutex
}

func (u *UserStore) Sync(b map[string]model.UserStats) error {
	const q = `
		UPDATE 
			users 
		SET 
		    announces = (announces + ?), 
		    uploaded = (uploaded + ?),
		    downloaded = (downloaded + ?)
		WHERE
			passkey = ?`
	// TODO use ctx for timeout
	ctx := context.Background()
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err := tx.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "Failed to prepare user Sync() tx")
	}
	for passkey, stats := range b {
		_, err := stmt.Exec(stats.Announces, stats.Uploaded, stats.Downloaded, passkey)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back user Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec user Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit user Sync() tx")
	}
	return nil
}

// Add will add a new user to the backing store
func (u *UserStore) Add(user model.User) error {
	if user.UserID > 0 {
		return errors.New("User already has a user_id")
	}
	const q = `
		INSERT INTO users 
		    (passkey, download_enabled, is_deleted) 
		VALUES
		    (?, ?, ?)`
	res, err := u.db.Exec(q, user.Passkey, true, false)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return errors.New("Failed to fetch insert ID")
	}
	user.UserID = uint32(lastID)
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetByPasskey(user *model.User, passkey string) error {
	const q = `SELECT * FROM users WHERE passkey = ?`
	if err := u.db.Get(user, q, passkey); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	return nil
}

// GetByID returns a user matching the userId
func (u *UserStore) GetByID(user *model.User, userID uint32) error {
	const q = `SELECT * FROM users WHERE user_id = ?`
	if err := u.db.Get(user, q, userID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by user_id")
	}
	return nil
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(user model.User) error {
	if user.UserID <= 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM users WHERE user_id = ?`
	if _, err := u.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (u *UserStore) Close() error {
	return u.db.Close()
}

type userDriver struct{}

// NewUserStore creates a new mysql backed user store.
func (ud userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &UserStore{
		db:      db,
		users:   map[string]model.User{},
		usersMx: sync.RWMutex{},
	}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
}

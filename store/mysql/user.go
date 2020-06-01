package mysql

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ErrNoResults is the string returned from the driver when no rows are returned
const ErrNoResults = "sql: no rows in result set"

// UserStore is the MySQL backed store.UserStore implementation
type UserStore struct {
	db *sqlx.DB
}

func (u *UserStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new UserStats provided
func (u *UserStore) Sync(b map[string]store.UserStats, cache *store.UserCache) error {
	const q = `CALL user_update_stats(?, ?, ?, ?)`
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
		_, err := stmt.Exec(passkey, stats.Announces, stats.Uploaded, stats.Downloaded)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back user Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec user Sync() tx")
		}
		if cache != nil {
			var usr store.User
			if cache.Get(&usr, passkey) {
				usr.Downloaded += stats.Downloaded
				usr.Uploaded += stats.Uploaded
				usr.Announces += stats.Announces
				cache.Set(usr)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit user Sync() tx")
	}

	return nil
}

// Add will add a new user to the backing store
func (u *UserStore) Add(user store.User) error {
	const q = `CALL user_add(?, ?, ?, ?, ?, ?, ?)`
	_, err := u.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetByPasskey(user *store.User, passkey string) error {
	const q = `CALL user_by_passkey(?)`
	if err := u.db.Get(user, q, passkey); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	return nil
}

// GetByID returns a user matching the userId
func (u *UserStore) GetByID(user *store.User, userID uint32) error {
	const q = `CALL user_by_id(?)`
	if err := u.db.Get(user, q, userID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by user_id")
	}
	return nil
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(user store.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `CALL user_delete(?)`
	if _, err := u.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

func (u *UserStore) Update(user store.User, oldPasskey string) error {
	const q = `CALL user_update(?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := u.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces,
		oldPasskey); err != nil {
		return errors.Wrapf(err, "Failed to update user")
	}
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (u *UserStore) Close() error {
	return u.db.Close()
}

type userDriver struct{}

// New creates a new mysql backed user store.
func (ud userDriver) New(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &UserStore{
		db: db,
	}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
}

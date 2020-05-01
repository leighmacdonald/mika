package mysql

import (
	"github.com/jmoiron/sqlx"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/store"
	"sync"
)

// UserStore is the MySQL backed store.UserStore implementation
type UserStore struct {
	db      *sqlx.DB
	users   map[string]model.User
	usersMx sync.RWMutex
}

// AddUser will add a new user to the backing store
func (u *UserStore) AddUser(_ *model.User) error {
	panic("implement me")
}

// GetUserByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetUserByPasskey(_ string) (*model.User, error) {
	return &model.User{}, nil
}

// GetUserByID returns a user matching the userId
func (u *UserStore) GetUserByID(_ uint32) (*model.User, error) {
	panic("implement me")
}

// DeleteUser removes a user from the backing store
func (u *UserStore) DeleteUser(_ *model.User) error {
	panic("implement me")
}

// Close will close the underlying database connection and clear the local caches
func (u *UserStore) Close() error {
	panic("implement me")
}

type userDriver struct{}

// NewUserStore creates a new mysql backed user store.
func (ud userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect("mysql", c.DSN())
	return &UserStore{
		db:      db,
		users:   map[string]model.User{},
		usersMx: sync.RWMutex{},
	}, nil
}

func init() {
	store.AddUserDriver("mysql", userDriver{})
}

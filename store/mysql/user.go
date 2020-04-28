package mysql

import (
	"github.com/jmoiron/sqlx"
	"mika/consts"
	"mika/model"
	"mika/store"
	"sync"
)

type UserStore struct {
	db      *sqlx.DB
	users   map[string]model.User
	usersMx sync.RWMutex
}

func (u *UserStore) GetUserByPasskey(passkey string) (model.User, error) {

}

func (u *UserStore) GetUserById(userId uint32) (model.User, error) {
	panic("implement me")
}

func (u *UserStore) DeleteUser(user model.User) error {
	panic("implement me")
}

func (u *UserStore) Close() error {
	panic("implement me")
}

type userDriver struct{}

func (ud userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*store.SQLConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	var db *sqlx.DB
	if c.Conn != nil {
		db = c.Conn
	} else {
		db = sqlx.MustConnect("mysql", c.DSN())
	}
	return &UserStore{
		db:      db,
		users:   map[string]model.User{},
		usersMx: sync.RWMutex{},
	}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
}

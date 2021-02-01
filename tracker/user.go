package tracker

import (
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
)

func Users() store.Users {
	return users
}

func UserAdd(user *store.User) error {
	if err := db.UserAdd(user); err != nil {
		return err
	}
	users[user.Passkey] = user
	return nil
}

func UserGetByPasskey(passkey string) (*store.User, error) {
	u, found := users[passkey]
	if !found {
		return nil, consts.ErrInvalidUser
	}
	return u, nil
}

func UserGetByUserID(userID uint32) (*store.User, error) {
	for _, u := range users {
		if u.UserID == userID {
			return u, nil
		}
	}
	return nil, consts.ErrInvalidUser
}

func UserGetByRemoteID(remoteID uint64) (*store.User, error) {
	for _, u := range users {
		if u.RemoteID == remoteID {
			return u, nil
		}
	}
	return nil, consts.ErrInvalidUser
}

func UserSave(user *store.User) error {
	return db.UserSave(user)
}

func UserSync(batch map[string]store.UserStats) error {
	if err := db.UserSync(batch); err != nil {
		return err
	}
	return nil
}

func UserDelete(user *store.User) error {
	// TODO remove from swarms
	// TODO updated references to deleted user?
	user.IsDeleted = true
	if err := UserSave(user); err != nil {
		return err
	}
	delete(users, user.Passkey)
	return nil
}

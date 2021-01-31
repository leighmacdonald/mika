package store

import (
	"time"
)

// User defines a basic user known to the tracker
// All users are considered enabled if they exist. You must remove them from the
// backing store to ensure they cannot access any resources
type User struct {
	UserID          uint32 `db:"user_id" json:"user_id"`
	RoleID          uint32 `json:"role_id" db:"role_id"`
	UserName        string `db:"user_name" json:"user_name"`
	Passkey         string `db:"passkey" json:"passkey"`
	IsDeleted       bool   `db:"is_deleted" json:"is_deleted"`
	DownloadEnabled bool   `db:"download_enabled" json:"download_enabled"`
	Downloaded      uint64 `db:"downloaded" json:"downloaded"`
	Uploaded        uint64 `db:"uploaded" json:"uploaded"`
	Announces       uint32 `db:"announces" json:"announces"`
	RemoteID        int64  `db:"remote_id" json:"remote_id"`
	Role            *Role  `json:"role" db:"-"`
}

// Valid performs basic validation of the user info ensuring we have the minimum required
// data to be considered valid by the tracker
func (u User) Valid() bool {
	return u.Passkey != "" && !u.IsDeleted
}

// Users is a slice of known users
type Users []*User
type Roles []*Role

// Remove removes a users from a Users slice
func (users Users) Remove(p *User) []*User {
	for i := len(users) - 1; i >= 0; i-- {
		if users[i].UserID == p.UserID {
			return append(users[:i], users[i+1:]...)
		}
	}
	return users
}

type Role struct {
	RoleID          uint32    `json:"role_id" db:"role_id"`
	RemoteID        int64     `json:"remote_id" db:"remote_id"`
	RoleName        string    `json:"role_name" db:"role_name"`
	Priority        int32     `json:"priority" db:"priority"`
	MultiUp         float64   `json:"multi_up" db:"multi_up"`
	MultiDown       float64   `json:"multi_down" db:"multi_down"`
	DownloadEnabled bool      `json:"download_enabled" db:"download_enabled"`
	UploadEnabled   bool      `json:"upload_enabled" db:"upload_enabled"`
	CreatedOn       time.Time `json:"created_on" db:"created_on"`
	UpdateOn        time.Time `json:"updated_on" db:"updated_on"`
}

package store

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// User defines a basic user known to the tracker
// All users are considered enabled if they exist. You must remove them from the
// backing store to ensure they cannot access any resources
type User struct {
	UserID          uint32    `db:"user_id" json:"user_id"`
	RoleID          uint32    `json:"role_id" db:"role_id"`
	RemoteID        uint64    `json:"remote_id" db:"remote_id"`
	UserName        string    `db:"user_name" json:"user_name"`
	Passkey         string    `db:"passkey" json:"passkey"`
	IsDeleted       bool      `db:"is_deleted" json:"is_deleted"`
	DownloadEnabled bool      `db:"download_enabled" json:"download_enabled"`
	Downloaded      uint64    `db:"downloaded" json:"downloaded"`
	Uploaded        uint64    `db:"uploaded" json:"uploaded"`
	Announces       uint32    `db:"announces" json:"announces"`
	CreatedOn       time.Time `db:"created_on" json:"created_on"`
	UpdatedOn       time.Time `db:"updated_on" json:"updated_on"`
	Role            *Role     `json:"role" db:"-"`

	// Keeps track of how often the values have been changes
	// TODO Items with the most writes will get written to soonest
	Writes uint32 `db:"-" json:"-"`
}

func (u User) Log() *log.Entry {
	return log.WithFields(log.Fields{"id": u.UserID, "name": u.UserName, "rid": u.RemoteID})
}

// Valid performs basic validation of the user info ensuring we have the minimum required
// data to be considered valid by the tracker
func (u User) Valid() bool {
	return u.Passkey != "" && !u.IsDeleted
}

// Users is a slice of known users
type Users map[string]*User

// Roles is a map of roles by role_id
type Roles map[uint32]*Role

// Remove removes a users from a Users slice
func (r Roles) Get(roleID uint32) *Role {
	return r[roleID]
}

// WhiteList is a map of whitelisted clients by 8 chars of client prefix
type WhiteList map[string]*WhiteListClient

// Remove removes a users from a Users slice
func (users Users) Remove(p *User) {
	delete(users, p.Passkey)
}

type Role struct {
	RoleID          uint32    `json:"role_id" db:"role_id"`
	RemoteID        uint64    `json:"remote_id" db:"remote_id"`
	RoleName        string    `json:"role_name" db:"role_name"`
	Priority        int32     `json:"priority" db:"priority"`
	MultiUp         float64   `json:"multi_up" db:"multi_up"`
	MultiDown       float64   `json:"multi_down" db:"multi_down"`
	DownloadEnabled bool      `json:"download_enabled" db:"download_enabled"`
	UploadEnabled   bool      `json:"upload_enabled" db:"upload_enabled"`
	CreatedOn       time.Time `json:"created_on" db:"created_on"`
	UpdatedOn       time.Time `json:"updated_on" db:"updated_on"`
}

func (r Role) Log() *log.Entry {
	return log.WithFields(log.Fields{"id": r.RoleID, "name": r.RoleName, "rid": r.RemoteID})
}

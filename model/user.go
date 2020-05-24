package model

// User defines a basic user known to the tracker
// All users are considered enabled if they exist. You must remove them from the
// backing store to ensure they cannot access any resources
type User struct {
	UserID          uint32 `db:"user_id" json:"user_id"`
	Passkey         string `db:"passkey" json:"passkey"`
	IsDeleted       bool   `db:"is_deleted" json:"is_deleted"`
	DownloadEnabled bool   `db:"download_enabled" json:"download_enabled"`
	Downloaded      uint64
	Uploaded        uint64
	Announces       uint32
}

// Valid performs basic validation of the user info ensuring we have the minimum required
// data to be considered valid by the tracker
func (u User) Valid() bool {
	return len(u.Passkey) == 20 && !u.IsDeleted
}

// Users is a slice of known users
type Users []User

// Remove removes a users from a Users slice
func (users Users) Remove(p User) []User {
	for i := len(users) - 1; i >= 0; i-- {
		if users[i] == p {
			return append(users[:i], users[i+1:]...)
		}
	}
	return users
}

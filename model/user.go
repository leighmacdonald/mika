package model

// User defines a basic user known to the tracker
// All users are considered enabled if they exist. You must remove them from the
// backing store to ensure they cannot access any resources
type User struct {
	UserID  uint32 `json:"user_id"`
	Passkey string `json:"passkey"`
}

// Valid performs basic validation of the user info ensuring we have the minimum required
// data to be considered valid by the tracker
func (u User) Valid() bool {
	return u.UserID > 0 && len(u.Passkey) == 20
}

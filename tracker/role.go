package tracker

import "github.com/leighmacdonald/mika/store"

func RoleAll() []*store.Role {
	var roleSet []*store.Role
	for _, r := range roles {
		roleSet = append(roleSet, r)
	}
	return roleSet
}

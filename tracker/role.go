package tracker

import (
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func RoleAll() []*store.Role {
	var roleSet []*store.Role
	for _, r := range roles {
		roleSet = append(roleSet, r)
	}
	return roleSet
}

func RoleDelete(roleID uint32) error {
	// TODO check user for dangling role references
	if err := db.RoleDelete(roleID); err != nil {
		return errors.Wrapf(err, "Failed to delete role")
	}
	delete(roles, roleID)
	log.WithField("role_id", roleID).Debug("Role deleted successfully")
	return nil
}

func RoleAdd(role *store.Role) error {
	if err := db.RoleSave(role); err != nil {
		return errors.Wrap(err, "Failed to save role")
	}
	roles[role.RoleID] = role
	role.Log().Debug("Role saved successfully")
	return nil
}

package cmd

import (
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// migrateCmd will migrate the database schema or install one if it does not exist
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate",
	Long:  `migrate`,
	Run: func(cmd *cobra.Command, args []string) {
		tracker.Init()
		if len(tracker.RoleAll()) == 0 {
			role := store.Role{
				RoleName:        "admin",
				Priority:        100,
				MultiUp:         1,
				MultiDown:       1,
				DownloadEnabled: true,
				UploadEnabled:   true,
				CreatedOn:       util.Now(),
				UpdateOn:        util.Now(),
			}
			if err := tracker.RoleAdd(&role); err != nil {
				log.Fatalf("Failed to save role: %v", err)
			}
			user := store.User{
				RoleID:          role.RoleID,
				UserName:        "admin",
				Passkey:         "mika",
				IsDeleted:       false,
				DownloadEnabled: true,
				CreatedOn:       util.Now(),
				UpdatedOn:       util.Now(),
			}
			if err := tracker.UserSave(&user); err != nil {
				log.Fatalf("Failed to save user: %v", err)
			}
		}

		log.Infof("Successfully migrated data store")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

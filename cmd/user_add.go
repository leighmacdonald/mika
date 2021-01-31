package cmd

import (
	"context"
	"github.com/leighmacdonald/mika/client"
	pb "github.com/leighmacdonald/mika/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	roleStr      = ""
	userAddParam = &pb.UserAddParams{}
)

// userAddCmd can be used to add users
var userAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a user to the tracker",
	Long:  `Add a user to the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleStr == "" {
			log.Fatal("role cannot be empty")
		}
		if userAddParam.UserName == "" {
			log.Fatal("role cannot be empty")
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Faile to connect to tracker")
		}
		_, err = client.UserAdd(context.Background(), userAddParam)
		if err != nil {
			log.Fatalf("Failed to add user: %v", err)
		}
		log.Infof("User added successfully")
	},
}

func init() {
	userCmd.AddCommand(userAddCmd)
	userAddCmd.Flags().StringVarP(&userAddParam.UserName, "name", "n", "", "Username of the user")
	userAddCmd.Flags().StringVarP(&userAddParam.Passkey, "passkey", "p", "", "Passkey for user. (default: random)")
	userAddCmd.Flags().BoolVarP(&userAddParam.DownloadEnabled, "download_enabled", "D", true, "Passkey for user. (default: true)")
	userAddCmd.Flags().Uint64VarP(&userAddParam.Downloaded, "downloaded", "d", 0, "User download total (default: 0)")
	userAddCmd.Flags().Uint64VarP(&userAddParam.Uploaded, "uploaded", "u", 0, "User upload total (default: 0)")
	userAddCmd.Flags().StringVarP(&roleStr, "role", "r", "", "User role")

}

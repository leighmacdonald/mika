package cmd

import (
	"context"
	"github.com/leighmacdonald/mika/client"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/rpc"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"strconv"
	"strings"
)

var (
	roleStr      = ""
	userAddParam = &pb.UserAddParams{}
)

// userCmd represents user admin commands
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "user commands",
	Long:  `user commands`,
}

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
			log.Fatal("user name cannot be empty")
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker")
			return
		}
		var validRoleID uint32
		roleID, err := strconv.ParseUint(roleStr, 10, 64)
		if err != nil {
			var roles []*store.Role
			rc, err := client.RoleAll(context.Background(), &emptypb.Empty{})
			if err != nil {
				log.Fatalf("Failed to get roles list")
				return
			}
			for {
				in, err := rc.Recv()
				if err == io.EOF {
					break
				}
				roles = append(roles, rpc.PBToRole(in))
			}
			for _, knownRole := range roles {
				if strings.ToLower(knownRole.RoleName) == strings.ToLower(roleStr) {
					validRoleID = knownRole.RoleID
				}
			}
		} else {
			validRoleID = uint32(roleID)
		}
		if validRoleID <= 0 {
			log.Fatalf("Failed to find a valid role_id")
			return
		}
		userAddParam.RoleId = validRoleID
		u, err2 := client.UserAdd(context.Background(), userAddParam)
		if err2 != nil {
			log.Fatalf("Failed to add user: %v", err2)
		}
		user := rpc.PBToUser(u)
		user.Log().Infof("User added successfully")
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userAddCmd.Flags().StringVarP(&userAddParam.UserName, "name", "n", "", "Username of the user")
	userAddCmd.Flags().StringVarP(&userAddParam.Passkey, "passkey", "p", "", "Passkey for user. (default: random)")
	userAddCmd.Flags().BoolVarP(&userAddParam.DownloadEnabled, "download_enabled", "D", true, "Passkey for user. (default: true)")
	userAddCmd.Flags().Uint64VarP(&userAddParam.Downloaded, "downloaded", "d", 0, "User download total (default: 0)")
	userAddCmd.Flags().Uint64VarP(&userAddParam.Uploaded, "uploaded", "u", 0, "User upload total (default: 0)")
	userAddCmd.Flags().StringVarP(&roleStr, "role", "r", "", "User role")
}

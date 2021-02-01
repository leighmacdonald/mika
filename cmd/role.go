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
)

var roleAddParam = &pb.RoleAddParams{}

// roleCmd represents role admin commands
var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "role commands",
	Long:  `role commands`,
}

// userAddCmd can be used to add users
var roleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a role to the tracker",
	Long:  `Add a role to the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleAddParam.RoleName == "" {
			log.Fatal("name cannot be empty")
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker")
			return
		}
		r, err2 := client.RoleAdd(context.Background(), roleAddParam)
		if err2 != nil {
			log.Fatalf("Failed to add new role: %v", err2)
		}
		role := rpc.PBToRole(r)
		role.Log().Infof("Role added successfully (id: %d, name: )", r.RoleId)
	},
}

// roleListCmd can be used to list roles
var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known roles",
	Long:  `List known roles`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker: %v", err)
			return
		}
		stream, err := client.RoleAll(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Fatalf("Failed to fetch roles: %v", err)
			return
		}
		var roles []*store.Role
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				break
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			roles = append(roles, rpc.PBToRole(in))
		}
		for _, r := range roles {
			log.Infof("Name: %s, ID: %d RemoteID: %d MultiUp %.1f MultiDn %.1f",
				r.RoleName, r.RoleID, r.RemoteID, r.MultiUp, r.MultiDown)
		}
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleAddCmd)

	roleAddCmd.Flags().StringVarP(&roleAddParam.RoleName, "name", "n", "", "Name of the role")
	roleAddCmd.Flags().Int32VarP(&roleAddParam.Priority, "priority", "p", 0, "Role Priority")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.DownloadEnabled, "download_enabled", "D", true, "Downloading enabled")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.UploadEnabled, "upload_enabled", "U", true, "Uploading enabled")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiDown, "multi_down", "d", 1.0, "Download multiplier")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiUp, "multi_up", "u", 1.0, "Upload multiplier")
}

package cmd

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/mika/client"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/rpc"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"os"
)

var (
	roleAddParam = &pb.RoleAddParams{}
	roleDelParam = &pb.RoleID{}
)

func defaultTable(title string) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if title != "" {
		t.SetTitle(title)
	}
	t.SetStyle(table.StyleColoredBright)
	return t
}

func renderRoles(roles []*store.Role, title string) {
	t := defaultTable(title)
	t.AppendHeader(table.Row{"role_id", "name", "priority", "xup", "xdn", "dl_enabled"})
	for _, role := range roles {
		t.AppendRow(table.Row{role.RoleID, role.RoleName, role.Priority, role.MultiDown,
			role.MultiUp, role.DownloadEnabled})
	}
	t.SortBy([]table.SortBy{{
		Name: "priority",
	}})
	t.Render()
}

// roleCmd represents role admin commands
var roleCmd = &cobra.Command{
	Use:               "role",
	Short:             "role commands",
	Long:              `role commands`,
	PersistentPreRunE: connectRPC,
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
		renderRoles([]*store.Role{role}, "Role added successfully")
		//role.Log().Infof("Role added successfully (id: %d, name: )", r.RoleId)
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
		renderRoles(roles, "List of all roles")
	},
}

// roleDeleteCmd can be used to delete roles
var roleDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a role from the tracker",
	Long:  `Delete a role from the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleDelParam.RoleId <= 0 && roleDelParam.RoleName == "" {
			log.Fatalf("Must supply one of: role name, role id")
			return
		}
		rid := &pb.RoleID{}
		if roleDelParam.RoleName != "" {
			rid.RoleName = roleDelParam.RoleName
		} else {
			rid.RoleId = roleDelParam.RoleId
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker")
			return
		}
		_, err2 := client.RoleDelete(context.Background(), rid)
		if err2 != nil {
			log.Fatalf("Failed to add new role: %v", err2)
		}
		log.Infof("Role deleted successfully")
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleAddCmd)
	roleCmd.AddCommand(roleDeleteCmd)

	roleDeleteCmd.Flags().StringVarP(&roleDelParam.RoleName, "name", "n", "", "Name of the role")
	roleDeleteCmd.Flags().Uint32VarP(&roleDelParam.RoleId, "id", "i", 0, "Role ID")

	roleAddCmd.Flags().StringVarP(&roleAddParam.RoleName, "name", "n", "", "Name of the role")
	roleAddCmd.Flags().Int32VarP(&roleAddParam.Priority, "priority", "p", 0, "Role Priority")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.DownloadEnabled, "download_enabled", "D", true, "Downloading enabled")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.UploadEnabled, "upload_enabled", "U", true, "Uploading enabled")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiDown, "multi_down", "d", 1.0, "Download multiplier")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiUp, "multi_up", "u", 1.0, "Upload multiplier")
}

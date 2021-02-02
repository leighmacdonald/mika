package cmd

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/rpc"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
)

var (
	wlParams = &pb.WhiteList{}
)

func renderWhitelist(wl []*store.WhiteListClient, title string) {
	t := defaultTable(title)
	t.AppendHeader(table.Row{"name", "prefix"})
	for _, w := range wl {
		t.AppendRow(table.Row{w.ClientName, w.ClientPrefix})
	}
	t.SortBy([]table.SortBy{{
		Name: "name",
	}})
	t.Render()
}

// whiteListCmd represents role admin commands
var whiteListCmd = &cobra.Command{
	Use:               "whitelist",
	Aliases:           []string{"wl"},
	Short:             "whitelist commands",
	Long:              `whitelist commands`,
	PersistentPreRunE: connectRPC,
}

// userAddCmd can be used to add users
var whiteListAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a client whitelist to the tracker",
	Long:  `Add a client whitelist to the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if wlParams.Name == "" || wlParams.Prefix == "" {
			log.Fatalf("Must supply non-empty name and prefix")
			return
		}
		if _, err := cl.WhiteListAdd(context.Background(), wlParams); err != nil {
			log.Fatalf("Failed to add whitelist entry")
			return
		}
		log.Infof("Client whitelist added: %s", wlParams.Name)
	},
}

// whiteListListCmd can be used to list roles
var whiteListListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known roles",
	Long:  `List known roles`,
	Run: func(cmd *cobra.Command, args []string) {
		stream, err := cl.WhiteListAll(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Fatalf("Failed to fetch wlc: %v", err)
			return
		}
		var wlc []*store.WhiteListClient
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				break
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			wlc = append(wlc, rpc.PBToWhiteList(in))
		}
		renderWhitelist(wlc, "List of all whitelisted clients")
	},
}

// whiteListDeleteCmd can be used to delete roles
var whiteListDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a client whitelist from the tracker",
	Long:  `Delete a client whitelist from the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(whiteListCmd)
	whiteListCmd.AddCommand(whiteListListCmd)
	whiteListCmd.AddCommand(whiteListAddCmd)
	whiteListCmd.AddCommand(whiteListDeleteCmd)

	whiteListAddCmd.Flags().StringVarP(&wlParams.Prefix, "prefix", "p", "", "Prefix to match the client")
	whiteListAddCmd.Flags().StringVarP(&wlParams.Name, "name", "n", "", "Name of the client")

	whiteListDeleteCmd.Flags().StringVarP(&wlParams.Prefix, "prefix", "p", "", "Prefix to match the client")
}

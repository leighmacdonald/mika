package cmd

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/model"
	"github.com/spf13/viper"
	"log"

	"github.com/spf13/cobra"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "CLI to administer a running instance",
	Long:  `CLI to administer a running instance`,
	//Run: func(cmd *cobra.Command, args []string) {
	//},
}

func newClient() *client.Client {
	host := viper.GetString(string(config.APIListen))
	return client.New(host)
}

// pingCmd represents the client command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Tests connecting to the backend tracker",
	Long:  "Tests connecting to the backend tracker",
	Run: func(cmd *cobra.Command, args []string) {
		if err := newClient().Ping(); err != nil {
			log.Fatalf("Could not connect to tracker")
		}
	},
}

// torrentCmd represents the base client torrent command set
var torrentCmd = &cobra.Command{
	Use:     "torrent",
	Aliases: []string{"t"},
	Short:   "Torrent administration related operations",
	Long:    "Torrent administration related operations",
}

var torrentDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del", "d"},
	Short:   "Delete a torrent from the tracker & store",
	Long:    "Delete a torrent from the tracker & store",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires at least 1 info_hash")
		}
		for _, ih := range args {
			if len(ih) != 40 {
				return fmt.Errorf("invalid info_hash: %s", ih)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		for _, hashString := range args {
			if err := c.TorrentDelete(model.InfoHashFromString(hashString)); err != nil {
				log.Fatalf("Error trying to delete %s: %s", hashString, err.Error())
			}
		}
	},
}

func init() {
	torrentCmd.AddCommand(torrentDeleteCmd)
	clientCmd.AddCommand(pingCmd)
	clientCmd.AddCommand(torrentCmd)
	rootCmd.AddCommand(clientCmd)
}

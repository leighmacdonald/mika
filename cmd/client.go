package cmd

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"

	"github.com/spf13/cobra"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:     "client",
	Short:   "CLI to administer a running instance",
	Long:    `CLI to administer a running instance`,
	Aliases: []string{"c"},
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

var torrentAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a"},
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
			p := strings.Split(hashString, ":")
			if len(p) != 2 {
				log.Fatalf(`Invalid format. Expected: <info_hash>:"release name"`)
			}
			if err := c.TorrentAdd(model.InfoHashFromString(p[0]), p[1]); err != nil {
				log.Fatalf("Err trying to delete %s: %s", hashString, err.Error())
			}
		}
	},
}

// torrentCmd represents the base client torrent command set
var userCmd = &cobra.Command{
	Use:     "user",
	Aliases: []string{"u"},
	Short:   "User administration related operations",
	Long:    "User administration related operations",
}

var userAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a"},
	Short:   "Add a user to the tracker & store",
	Args:    cobra.MaximumNArgs(1),
	Long:    "Add a user to the tracker & store",
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		passkey := cmd.Flag("passkey").Value.String()
		if passkey == "" {
			passkey = util.NewPasskey()
		}
		if err := c.UserAdd(passkey); err != nil {
			log.Errorf("Error adding user: %s", err.Error())
		}
		log.Infof("Added user with passkey: %s", passkey)
	},
}

var userDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del", "d"},
	Short:   "Delete a user from the tracker & store",
	Long:    "Delete a user from the tracker & store",
	//Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		passkey := cmd.Flag("passkey").Value.String()
		if passkey == "" {
			log.Fatalf("Invalid passkey")
		}
		if err := newClient().UserDelete(passkey); err != nil {
			log.Errorf("Failed to remove user: %s", err.Error())
		}
	},
}

func init() {
	userAddCmd.PersistentFlags().StringP("passkey", "p", "", "User Passkey")
	userDeleteCmd.PersistentFlags().StringP("passkey", "p", "", "User Passkey")

	torrentCmd.AddCommand(torrentAddCmd)
	torrentCmd.AddCommand(torrentDeleteCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userDeleteCmd)
	clientCmd.AddCommand(pingCmd)
	clientCmd.AddCommand(torrentCmd)
	clientCmd.AddCommand(userCmd)
	rootCmd.AddCommand(clientCmd)
}

package cmd

import (
	"fmt"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path/filepath"
	"strconv"
	"strings"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:     "client",
	Short:   "CLI to administer a running instance",
	Long:    `CLI to administer a running instance`,
	Aliases: []string{"c"},
}

func newClient() *client.Client {
	host := config.GetString(config.APIListen)
	key := config.GetString(config.APIKey)
	if strings.HasPrefix(host, ":") {
		host = "http://localhost" + host
	}
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return client.New(host, key)
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
		var ih store.InfoHash
		for _, hashString := range args {
			if err := store.InfoHashFromString(&ih, hashString); err != nil {
				log.Fatalf("Error trying to parse infohash %s: %s", hashString, err.Error())
			}
			if err := c.TorrentDelete(ih); err != nil {
				log.Fatalf("Error trying to delete %s: %s", hashString, err.Error())
			}
		}
	},
}

var torrentAddFileCmd = &cobra.Command{
	Use:     "addfile",
	Aliases: []string{"af"},
	Short:   "Add a torrent from a file",
	Long:    "Add a torrent from a file",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires at least 1 torrent file")
		}
		for _, arg := range args {
			if !util.Exists(arg) {
				return errors.Errorf("Unable to find file: %s", arg)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		var infoHash store.InfoHash
		for _, fileName := range args {
			mi, err := metainfo.LoadFromFile(fileName)
			if err != nil {
				return
			}
			if err := store.InfoHashFromHex(&infoHash, mi.HashInfoBytes().HexString()); err != nil {
				log.Fatalf(err.Error())
			}
			basename := filepath.Base(fileName)
			name := strings.TrimSuffix(basename, filepath.Ext(basename))
			if err := c.TorrentAdd(infoHash, basename); err != nil {
				log.Fatalf("Error trying to add %s: %s", name, err.Error())
			}
		}
	},
}

var torrentAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a"},
	Short:   "Add a torrent to the tracker & store",
	Long:    "Add a torrent to the tracker & store",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires at least 1 info_hash")
		}
		for _, ih := range args {
			p := strings.SplitN(ih, ":", 2)
			if len(p) != 2 {
				log.Fatalf(`Invalid format. Expected: <info_hash>:"release name"`)
			}
			if len(p[0]) != 40 {
				return fmt.Errorf("invalid info_hash: %s", ih)
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		var infoHash store.InfoHash
		for _, hashString := range args {
			p := strings.SplitN(hashString, ":", 2)
			if len(p) != 2 {
				log.Fatalf(`Invalid format. Expected: <info_hash>:"release name"`)
			}
			if err := store.InfoHashFromString(&infoHash, p[0]); err != nil {
				log.Fatalf(err.Error())
			}
			if err := c.TorrentAdd(infoHash, p[1]); err != nil {
				log.Fatalf("Error trying to add %s: %s", hashString, err.Error())
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
	Args:    cobra.MaximumNArgs(2),
	Long:    "Add a user to the tracker & store",
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient()
		userId := cmd.Flag("id").Value.String()
		if userId == "" {
			log.Fatalf("Must set user id to positive integer")
		}
		idVal, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			log.Fatalf("Must set user id to positive integer")
		}
		passkey := cmd.Flag("passkey").Value.String()
		if passkey == "" {
			passkey = util.NewPasskey()
		}
		var user store.User
		user.Passkey = passkey
		user.UserID = uint32(idVal)
		if err := c.UserAdd(user); err != nil {
			log.Fatalf("Error adding user: %s", err.Error())
		}
		log.Infof("Added user (%d) with passkey: %s", idVal, passkey)
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
	userAddCmd.PersistentFlags().StringP("id", "u", "", "Your internal user ID")
	userDeleteCmd.PersistentFlags().StringP("passkey", "p", "", "User Passkey")

	torrentCmd.AddCommand(torrentAddCmd)
	torrentCmd.AddCommand(torrentAddFileCmd)
	torrentCmd.AddCommand(torrentDeleteCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userDeleteCmd)
	clientCmd.AddCommand(pingCmd)
	clientCmd.AddCommand(torrentCmd)
	clientCmd.AddCommand(userCmd)
	rootCmd.AddCommand(clientCmd)
}

package cmd

import (
	"context"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the tracker and serve requests",
	Long:  `Start the tracker and serve requests`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		opts := tracker.NewDefaultOpts()
		opts.GeodbEnabled = viper.GetBool(string(config.GeodbEnabled))
		opts.BatchInterval = viper.GetDuration(string(config.TrackerBatchUpdateInterval))
		opts.ReaperInterval = viper.GetDuration(string(config.TrackerReaperInterval))
		opts.AnnInterval = viper.GetDuration(string(config.TrackerAnnounceInterval))
		opts.AnnIntervalMin = viper.GetDuration(string(config.TrackerAnnounceIntervalMin))
		opts.AllowNonRoutable = viper.GetBool(string(config.TrackerAllowNonRoutable))

		runMode := viper.GetString(string(config.GeneralRunMode))
		ts, err := store.NewTorrentStore(
			viper.GetString(string(config.StoreTorrentType)),
			config.GetStoreConfig(config.Torrent))
		if err != nil {
			log.Fatalf("Failed to setup torrent store: %s", err)
		}
		opts.Torrents = ts
		p, err2 := store.NewPeerStore(viper.GetString(string(config.StorePeersType)),
			config.GetStoreConfig(config.Peers))
		if err2 != nil {
			log.Fatalf("Failed to setup peer store: %s", err2)
		}
		opts.Peers = p
		u, err3 := store.NewUserStore(viper.GetString(string(config.StoreUsersType)),
			config.GetStoreConfig(config.Users))
		if err3 != nil {
			log.Fatalf("Failed to setup user store: %s", err3)
		}
		opts.Users = u
		var geodb geo.Provider
		if config.GetBool(config.GeodbEnabled) {
			geodb = geo.New(config.GetString(config.GeodbPath), runMode == "release")
		} else {
			geodb = &geo.DummyProvider{}
		}
		opts.Geodb = geodb
		tkr, err4 := tracker.New(ctx, opts)
		if err4 != nil {
			log.Fatalf("Failed to initialize tracker: %s", err)
		}

		var infoHash model.InfoHash
		mi, err5 := metainfo.LoadFromFile("examples/data/demo_torrent_data.torrent")
		if err5 != nil {
			return
		}
		if err := model.InfoHashFromHex(&infoHash, mi.HashInfoBytes().HexString()); err != nil {
			log.Fatalf(err.Error())
		}
		if err := tkr.Torrents.Add(model.Torrent{
			ReleaseName: "demo torrent",
			InfoHash:    infoHash,
		}); err != nil {
			panic("bad torrent")
		}
		if err := tkr.Users.Add(model.User{
			UserID:          1,
			Passkey:         "01234567890123456789",
			IsDeleted:       false,
			DownloadEnabled: true,
		}); err != nil {
			panic("bad user")
		}

		btOpts := tracker.DefaultHTTPOpts()
		btOpts.ListenAPI = viper.GetString(string(config.TrackerListen))
		btOpts.ListenAPITLS = viper.GetBool(string(config.TrackerTLS))
		btOpts.Handler = tracker.NewBitTorrentHandler(tkr)
		btServer := tracker.CreateServer(btOpts)

		apiOpts := tracker.DefaultHTTPOpts()
		apiOpts.ListenAPI = viper.GetString(string(config.APIListen))
		apiOpts.ListenAPITLS = viper.GetBool(string(config.APITLS))
		apiOpts.Handler = tracker.NewAPIHandler(tkr)
		apiServer := tracker.CreateServer(apiOpts)

		go tkr.PeerReaper()
		go tkr.StatWorker()

		go func() {
			if err := btServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()

		go func() {
			if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()

		util.WaitForSignal(ctx, func(ctx context.Context) error {
			if err := apiServer.Shutdown(ctx); err != nil {
				log.Fatalf("Error closing servers gracefully; %s", err)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

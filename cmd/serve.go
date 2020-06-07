package cmd

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	"github.com/spf13/cobra"
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
		opts.GeodbEnabled = config.GetBool(config.GeodbEnabled)
		opts.BatchInterval = config.GetDuration(config.TrackerBatchUpdateInterval)
		opts.ReaperInterval = config.GetDuration(config.TrackerReaperInterval)
		opts.AnnInterval = config.GetDuration(config.TrackerAnnounceInterval)
		opts.AnnIntervalMin = config.GetDuration(config.TrackerAnnounceIntervalMin)
		opts.AllowNonRoutable = config.GetBool(config.TrackerAllowNonRoutable)
		opts.AutoRegister = config.GetBool(config.TrackerAutoRegister)
		ts, err := store.NewTorrentStore(
			config.GetString(config.StoreTorrentType),
			config.GetStoreConfig(config.Torrent))
		if err != nil {
			log.Fatalf("Failed to setup torrent store: %s", err)
		}
		opts.Torrents = ts
		p, err2 := store.NewPeerStore(config.GetString(config.StorePeersType),
			config.GetStoreConfig(config.Peers))
		if err2 != nil {
			log.Fatalf("Failed to setup peer store: %s", err2)
		}
		opts.Peers = p
		u, err3 := store.NewUserStore(config.GetString(config.StoreUsersType),
			config.GetStoreConfig(config.Users))
		if err3 != nil {
			log.Fatalf("Failed to setup user store: %s", err3)
		}
		opts.Users = u
		var geodb geo.Provider
		if config.GetBool(config.GeodbEnabled) {
			geodb, err = geo.New(config.GetString(config.GeodbPath))
			if err != nil {
				log.Fatalf("Could not validate geo database")
			}
		} else {
			geodb = &geo.DummyProvider{}
		}
		opts.Geodb = geodb
		tkr, err4 := tracker.New(ctx, opts)
		if err4 != nil {
			log.Fatalf("Failed to initialize tracker: %s", err)
		}
		_ = tkr.LoadWhitelist()

		btOpts := tracker.DefaultHTTPOpts()
		btOpts.ListenAddr = config.GetString(config.TrackerListen)
		btOpts.UseTLS = config.GetBool(config.TrackerTLS)
		btOpts.Handler = tracker.NewBitTorrentHandler(tkr)
		btServer := tracker.NewHTTPServer(btOpts)

		apiOpts := tracker.DefaultHTTPOpts()
		apiOpts.ListenAddr = config.GetString(config.APIListen)
		apiOpts.UseTLS = config.GetBool(config.APITLS)
		apiOpts.Handler = tracker.NewAPIHandler(tkr)
		apiServer := tracker.NewHTTPServer(apiOpts)

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

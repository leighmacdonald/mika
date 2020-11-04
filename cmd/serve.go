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
		opts.GeodbEnabled = config.GeoDB.Enabled
		opts.BatchInterval = config.Tracker.BatchUpdateIntervalParsed
		opts.ReaperInterval = config.Tracker.ReaperIntervalParsed
		opts.AnnInterval = config.Tracker.AnnounceIntervalParsed
		opts.AnnIntervalMin = config.Tracker.AnnounceIntervalMinimumParsed
		opts.AllowNonRoutable = config.Tracker.AllowNonRoutable
		opts.AutoRegister = config.Tracker.AutoRegister
		opts.Public = config.Tracker.Public
		opts.TorrentCacheEnabled = config.TorrentStore.Cache
		opts.PeerCacheEnabled = config.PeerStore.Cache
		opts.UserCacheEnabled = config.UserStore.Cache
		ts, err := store.NewTorrentStore(config.TorrentStore)
		if err != nil {
			log.Fatalf("Failed to setup torrent store: %s", err)
		}
		opts.Torrents = ts
		p, err2 := store.NewPeerStore(config.PeerStore)
		if err2 != nil {
			log.Fatalf("Failed to setup peer store: %s", err2)
		}
		opts.Peers = p
		u, err3 := store.NewUserStore(config.UserStore)
		if err3 != nil {
			log.Fatalf("Failed to setup user store: %s", err3)
		}
		opts.Users = u
		var geodb geo.Provider
		if config.GeoDB.Enabled {
			geodb, err = geo.New(config.GeoDB.Path)
			if err != nil {
				log.Fatalf("Could not validate geo database. You may need to run ./mika updategeo")
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
		btOpts.ListenAddr = config.Tracker.Listen
		btOpts.UseTLS = config.Tracker.TLS
		btOpts.Handler = tracker.NewBitTorrentHandler(tkr)
		btServer := tracker.NewHTTPServer(btOpts)

		apiOpts := tracker.DefaultHTTPOpts()
		apiOpts.ListenAddr = config.API.Listen
		apiOpts.UseTLS = config.API.TLS
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

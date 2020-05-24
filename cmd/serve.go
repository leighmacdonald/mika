package cmd

import (
	"context"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/model"
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
		tkr, err := tracker.New(ctx)
		if err != nil {
			log.Fatalf("Failed to initialize tracker: %s", err)
		}

		var infoHash model.InfoHash
		mi, err := metainfo.LoadFromFile("examples/data/demo_torrent_data.torrent")
		if err != nil {
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
		listenBT := viper.GetString(string(config.TrackerListen))
		listenBTTLS := viper.GetBool(string(config.TrackerTLS))
		btHandler := tracker.NewBitTorrentHandler(tkr)
		btServer := tracker.CreateServer(btHandler, listenBT, listenBTTLS)

		listenAPI := viper.GetString(string(config.APIListen))
		listenAPITLS := viper.GetBool(string(config.APITLS))
		apiHandler := tracker.NewAPIHandler(tkr)
		apiServer := tracker.CreateServer(apiHandler, listenAPI, listenAPITLS)

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

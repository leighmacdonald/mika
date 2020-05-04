package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"mika/config"
	h "mika/http"
	"mika/tracker"
	"mika/util"
	"net/http"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the tracker and serve requests",
	Long:  `Start the tracker and serve requests`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		tkr, err := tracker.New()
		if err != nil {
			log.Fatalf("Failed to initialize tracker: %s", err)
		}
		listenBT := viper.GetString(string(config.TrackerListen))
		listenBTTLS := viper.GetBool(string(config.TrackerTLS))
		btHandler := h.NewBitTorrentHandler(tkr)
		btServer := h.CreateServer(btHandler, listenBT, listenBTTLS)

		listenAPI := viper.GetString(string(config.APIListen))
		listenAPITLS := viper.GetBool(string(config.APITLS))
		apiHandler := h.NewAPIHandler(tkr)
		apiServer := h.CreateServer(apiHandler, listenAPI, listenAPITLS)

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

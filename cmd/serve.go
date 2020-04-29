/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"mika/config"
	h "mika/http"
	"mika/tracker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
		listenBT := viper.GetString(config.TrackerListen)
		listenBTTLS := viper.GetBool(config.TrackerTLS)
		btHandler := h.NewBitTorrentHandler(tkr)
		btServer := h.CreateServer(btHandler, listenBT, listenBTTLS)

		listenAPI := viper.GetString(config.APIListen)
		listenAPITLS := viper.GetBool(config.APITLS)
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
		//go h.StartListeners([]*http.Server{btServer, apiServer})
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-sigChan:
			c, _ := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
			if err := apiServer.Shutdown(c); err != nil {
				log.Fatalf("Error closing servers gracefully; %s", err)
			}
		}

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

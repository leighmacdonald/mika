package cmd

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/geo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

// updategeoCmd represents the updategeo command
var updategeoCmd = &cobra.Command{
	Use:   "updategeo",
	Short: "Downloaded the latest geo database from MaxMind.com",
	Long:  `Downloaded the latest geo database from MaxMind.com`,
	Run: func(cmd *cobra.Command, args []string) {
		t0 := time.Now()
		log.Infof("Starting download of ip2location databases")
		if err := geo.DownloadDB(config.GeoDB.Path, config.GeoDB.APIKey); err != nil {
			log.Errorf("failed to download database: %s", err.Error())
		} else {
			d := time.Since(t0).String()
			log.Infof("Successfully downloaded geoip db to: %s (%s)", config.GeoDB.Path, d)
		}
	},
}

func init() {
	rootCmd.AddCommand(updategeoCmd)
}

/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"mika/geo"
)

// updategeoCmd represents the updategeo command
var updategeoCmd = &cobra.Command{
	Use:   "updategeo",
	Short: "Downloaded the latest geo database from MaxMind.com",
	Long:  `Downloaded the latest geo database from MaxMind.com`,
	Run: func(cmd *cobra.Command, args []string) {
		key := viper.GetString("geodb_api_key")
		outPath := viper.GetString("geodb_path")
		if err := geo.DownloadDB(outPath, key); err != nil {
			log.Errorf("failed to download database: %s", err.Error())
		} else {
			log.Infof("Successfully downloaded geoip db to: %s", outPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(updategeoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updategeoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updategeoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

package cmd

import (
	"fmt"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
// TODO remove build time for easier reproducible builds?
var rootCmd = &cobra.Command{
	Use:     "mika",
	Short:   "",
	Long:    ``,
	Version: fmt.Sprintf("mika (git:%s) (date:%s)", consts.BuildVersion, consts.BuildTime),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}

func init() {
	cobra.OnInitialize(func() {
		if err := config.Read(cfgFile); err != nil {
			log.Fatalf("Could not load & parse config: %v", err)
		}
	})

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./mika.yaml)")
}

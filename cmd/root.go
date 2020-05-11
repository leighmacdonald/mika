package cmd

import (
	"fmt"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/spf13/cobra"
	"log"
	"os"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		if err := config.Read(cfgFile); err != nil {
			log.Fatal("Could not load config")
		}
	})

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./mika.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

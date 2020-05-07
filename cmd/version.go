/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/leighmacdonald/mika/consts"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the current version or build hash",
	Long:  `Get the current version or build hash`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mika (git:%s) (date:%s)", consts.BuildVersion, consts.BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

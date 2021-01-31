package cmd

import (
	"github.com/spf13/cobra"
)

// userCmd represents user admin commands
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "user commands",
	Long:  `user commands`,
}

func init() {
	rootCmd.AddCommand(userCmd)
}

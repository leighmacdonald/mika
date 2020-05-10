package cmd

import (
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/config"
	"github.com/spf13/viper"
	"log"

	"github.com/spf13/cobra"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "CLI to administer a running instance",
	Long:  `CLI to administer a running instance`,
	//Run: func(cmd *cobra.Command, args []string) {
	//},
}

// pingCmd represents the client command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "CLI to administer a running instance",
	Long:  `CLI to administer a running instance`,
	Run: func(cmd *cobra.Command, args []string) {
		host := viper.GetString(string(config.APIListen))
		c := client.New(host)
		if err := c.Ping(); err != nil {
			log.Fatalf("Could not connect to tracker")
		}
	},
}

func init() {
	clientCmd.AddCommand(pingCmd)
	rootCmd.AddCommand(clientCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clientCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clientCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// +build demos

package cmd

import (
	"context"
	"github.com/leighmacdonald/mika/examples/exampleapi"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
)

// demoapiCmd represents the demoapi command
var demoapiCmd = &cobra.Command{
	Use:   "demoapi",
	Short: "A example implementation of a HTTP backed store",
	Long:  `A example implementation of a HTTP backed store`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		e := exampleapi.New()
		go func() {
			if err := e.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()
		util.WaitForSignal(ctx, func(ctx context.Context) error {
			if err := e.Shutdown(ctx); err != nil {
				log.Fatalf("Error closing servers gracefully; %s", err)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(demoapiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// demoapiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// demoapiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

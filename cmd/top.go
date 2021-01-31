package cmd

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// topCmd rune a top like status display of the running tracker
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "A top like status display of the running tracker",
	Long:  `A top like status display of the running tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := ui.Init(); err != nil {
			log.Fatalf("failed to initialize termui: %v", err)
		}
		defer ui.Close()

		p := widgets.NewParagraph()
		p.Text = "Hello World!"
		p.SetRect(0, 0, 25, 5)

		ui.Render(p)

		for e := range ui.PollEvents() {
			if e.Type == ui.KeyboardEvent {
				break
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
}

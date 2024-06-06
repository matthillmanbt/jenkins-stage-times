package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(openCmd)
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a build in your browser",
	Long:  `Open a build log in your browser and also print the URL.`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/job/%s/%s/flowGraphTable", viper.Get("host"), viper.Get("pipeline"), args[0])

		style := stdRe.NewStyle().
			Bold(true)

		fmt.Println(style.Render(url))

		command := "open"
		if runtime.GOOS == "windows" {
			command = "start"
		}
		return Spawn(command, url).Wait()
	},
}

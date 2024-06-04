package cmd

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	buildIDs []string
)

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringArrayVarP(&buildIDs, "build", "b", []string{}, "Build ID")
	statusCmd.MarkFlagRequired("build")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print the status of a given build ID",
	Long:  `Query Jenkins for the status of a build build given the build ID`,
	RunE: func(cmd *cobra.Command, args []string) error {
		builds := []*WorkflowRun{}
		for _, buildID := range buildIDs {
			build, err := getBuildInfo(buildID)
			if err != nil {
				verbose("getBuildInfo returned error")
				return err
			}
			builds = append(builds, build)
		}

		sort.Slice(builds, func(i, j int) bool {
			return builds[i].ID < builds[j].ID
		})

		style := stdRe.NewStyle().
			Foreground(lipgloss.NoColor{})
		infoStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(orange)
		errStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(white).
			Background(red)
		successStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(white).
			Background(green)

		pipeline := infoStyle.Render(viper.GetString("pipeline"))
		for _, build := range builds {
			id := infoStyle.Render(build.ID)
			name := infoStyle.Render(build.DisplayName)
			rStyle := infoStyle
			if build.Result == "SUCCESS" {
				rStyle = successStyle
			} else if build.Result == "FAILURE" {
				rStyle = errStyle
			}
			result := rStyle.Render(build.Result)
			fmt.Println(style.Render(fmt.Sprintf("%s: The status for [%s] on branch [%s] is [%s]", id, name, pipeline, result)))
		}

		return nil
	},
}

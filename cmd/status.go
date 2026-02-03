package cmd

import (
	"fmt"
	"sort"

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

		pipeline := infoBoldStyle.Render(viper.GetString("pipeline"))
		for _, build := range builds {
			id := infoBoldStyle.Render(build.ID)
			name := infoBoldStyle.Render(build.DisplayName)
			rStyle := infoBoldStyle
			if build.Result == "SUCCESS" {
				rStyle = successStyle
			} else if build.Result == "FAILURE" {
				rStyle = failureStyle
			}
			result := rStyle.Render(build.Result)
			fmt.Println(noStyle.Render(fmt.Sprintf("%s: The status for [%s] on branch [%s] is [%s]", id, name, pipeline, result)))
		}

		return nil
	},
}

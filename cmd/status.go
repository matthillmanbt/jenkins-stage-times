package cmd

import (
	"fmt"
	"jenkins/internal/jenkins"
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
}

var statusCmd = &cobra.Command{
	Use:   "status [build_id] [...build_id]",
	Short: "Print the status of a given build ID",
	Long: `Query Jenkins for the status of a build given the build ID.

Build IDs can be passed as positional arguments or with the -b flag:
  jenkins status 1234 5678
  jenkins status -b 1234 -b 5678`,
	RunE: func(cmd *cobra.Command, args []string) error {
		allIDs := append(buildIDs, args...)
		if len(allIDs) == 0 {
			return fmt.Errorf("at least one build ID is required (as argument or with -b flag)")
		}

		builds := []*jenkins.WorkflowRun{}
		for _, buildID := range allIDs {
			build, err := jenkinsClient.GetBuildInfo(viper.GetString("pipeline"), buildID)
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

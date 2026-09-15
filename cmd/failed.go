package cmd

import (
	"fmt"
	"jenkins/internal/formatting"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(failedCmd)
}

var failedCmd = &cobra.Command{
	Use:   "failed [build_id]",
	Short: "List all failed stages in a build",
	Long:  `Show all stages that failed in a given build, with their IDs and durations for further investigation.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		buildID := args[0]

		// Get job details with stages
		job, err := jenkinsClient.GetJobDetails(viper.GetString("pipeline"), buildID)
		if err != nil {
			return fmt.Errorf("failed to get build details: %w", err)
		}

		// Find failed leaf stages with their paths
		var failedLeaves []StageWithPath
		collectLeafStages(job.Stages, nil, keepFailedLeaf, &failedLeaves)

		if len(failedLeaves) == 0 {
			fmt.Println(successStyle.Render("✓ No failed stages found"))
			return nil
		}

		// Print header
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("Failed stages in build %s:", buildID)))
		fmt.Println()

		// Print each failed stage with full path
		for _, item := range failedLeaves {
			fullPath := strings.Join(append(item.Path, item.Stage.Name), " > ")
			duration := formatting.Duration(time.Duration(item.Stage.Duration * int(time.Millisecond)))
			fmt.Printf("  %s %s\n",
				failureStyle.Render("✗"),
				infoBoldStyle.Render(fullPath))
			fmt.Printf("    ID:       %s\n", item.Stage.ID)
			fmt.Printf("    Status:   %s\n", item.Stage.Status)
			fmt.Printf("    Duration: %s\n", duration)
			fmt.Printf("    Node:     %s\n", item.Stage.ExecNode)
			fmt.Printf("    Log URL:  %s\n", item.Stage.Links.Log.HREF)
			fmt.Println()
		}

		fmt.Println(grayStyle.Render(fmt.Sprintf("Total failed stages: %d", len(failedLeaves))))
		fmt.Println(grayStyle.Render(fmt.Sprintf("Use 'jenkins stage-log %s <stage_id>' to view logs", buildID)))

		return nil
	},
}

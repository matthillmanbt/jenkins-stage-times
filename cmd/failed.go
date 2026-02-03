package cmd

import (
	"fmt"
	"jenkins/internal/formatting"
	"jenkins/internal/jenkins"
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

		// Find failed stages
		var failedStages []jenkins.Stage
		collectFailedStages(job.Stages, &failedStages)

		if len(failedStages) == 0 {
			fmt.Println(successStyle.Render("✓ No failed stages found"))
			return nil
		}

		// Print header
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("Failed stages in build %s:", buildID)))
		fmt.Println()

		// Print each failed stage
		for _, stage := range failedStages {
			duration := formatting.Duration(time.Duration(stage.Duration * int(time.Millisecond)))
			fmt.Printf("  %s %s\n",
				failureStyle.Render("✗"),
				infoBoldStyle.Render(stage.Name))
			fmt.Printf("    ID:       %s\n", stage.ID)
			fmt.Printf("    Status:   %s\n", stage.Status)
			fmt.Printf("    Duration: %s\n", duration)
			fmt.Printf("    Node:     %s\n", stage.ExecNode)
			fmt.Println()
		}

		fmt.Println(grayStyle.Render(fmt.Sprintf("Total failed stages: %d", len(failedStages))))
		fmt.Println(grayStyle.Render(fmt.Sprintf("Use 'jenkins stage-log %s <stage_id>' to view logs", buildID)))

		return nil
	},
}

// collectFailedStages recursively collects all failed stages including nested ones
func collectFailedStages(stages []jenkins.Stage, failed *[]jenkins.Stage) {
	for _, stage := range stages {
		if stage.Status == "FAILED" || stage.Status == "ABORTED" {
			*failed = append(*failed, stage)
		}
		// Recursively check nested stages
		if len(stage.StageFlowNodes) > 0 {
			collectFailedStages(stage.StageFlowNodes, failed)
		}
	}
}

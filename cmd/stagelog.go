package cmd

import (
	"fmt"
	"jenkins/internal/jenkins"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	tailLines int
	headLines int
)

func init() {
	rootCmd.AddCommand(stageLogCmd)

	stageLogCmd.Flags().IntVarP(&tailLines, "tail", "t", 0, "Show only the last N lines")
	stageLogCmd.Flags().IntVarP(&headLines, "head", "n", 0, "Show only the first N lines")
}

var stageLogCmd = &cobra.Command{
	Use:   "stage-log [build_id] [stage_id]",
	Short: "Get the console log for a specific stage",
	Long:  `Fetch and display the console output for a specific stage in a build. Useful for debugging failed stages.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		buildID := args[0]
		stageID := args[1]

		// Get job details to find the stage
		job, err := jenkinsClient.GetJobDetails(viper.GetString("pipeline"), buildID)
		if err != nil {
			return fmt.Errorf("failed to get build details: %w", err)
		}

		// Find the stage
		stage := findStageByID(job.Stages, stageID)
		if stage == nil {
			return fmt.Errorf("stage %s not found in build %s", stageID, buildID)
		}

		// Print stage info header
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("Stage: %s", stage.Name)))
		fmt.Println(grayStyle.Render(fmt.Sprintf("Build: %s | Stage ID: %s | Status: %s",
			buildID, stageID, stage.Status)))
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println()

		// Get the log
		if stage.Links.Log.HREF == "" {
			return fmt.Errorf("no log available for this stage")
		}

		node, err := jenkinsClient.GetStageLog(stage.Links.Log.HREF)
		if err != nil {
			return fmt.Errorf("failed to get stage log: %w", err)
		}

		// Process output based on flags
		output := node.Text
		lines := strings.Split(output, "\n")

		if tailLines > 0 && tailLines < len(lines) {
			lines = lines[len(lines)-tailLines:]
			fmt.Println(grayStyle.Render(fmt.Sprintf("(showing last %d lines)", tailLines)))
		} else if headLines > 0 && headLines < len(lines) {
			lines = lines[:headLines]
			fmt.Println(grayStyle.Render(fmt.Sprintf("(showing first %d lines)", headLines)))
		}

		// Print the log
		for _, line := range lines {
			fmt.Println(line)
		}

		return nil
	},
}

// findStageByID recursively searches for a stage by ID
func findStageByID(stages []jenkins.Stage, stageID string) *jenkins.Stage {
	for _, stage := range stages {
		if stage.ID == stageID {
			return &stage
		}
		// Search nested stages
		if len(stage.StageFlowNodes) > 0 {
			if found := findStageByID(stage.StageFlowNodes, stageID); found != nil {
				return found
			}
		}
	}
	return nil
}

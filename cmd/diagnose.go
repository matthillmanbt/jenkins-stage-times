package cmd

import (
	"fmt"
	"jenkins/internal/formatting"
	"jenkins/internal/jenkins"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	showAllStages bool
	maxLogLines   int
)

func init() {
	rootCmd.AddCommand(diagnoseCmd)

	diagnoseCmd.Flags().BoolVarP(&showAllStages, "all", "a", false, "Show all stages, not just failed ones")
	diagnoseCmd.Flags().IntVarP(&maxLogLines, "log-lines", "l", 50, "Maximum lines of log to show per stage (0 for all)")
}

type StageWithPath struct {
	Stage jenkins.Stage
	Path  []string
}

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [build_id]",
	Short: "Analyze a build and show failed stages with logs",
	Long: `Comprehensive build analysis that shows:
  - Build status and duration
  - All failed stages with their logs
  - Summary of issues for AI analysis`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		buildID := args[0]

		// Get build info
		buildInfo, err := jenkinsClient.GetBuildInfo(viper.GetString("pipeline"), buildID)
		if err != nil {
			return fmt.Errorf("failed to get build info: %w", err)
		}

		// Get job details with stages
		job, err := jenkinsClient.GetJobDetails(viper.GetString("pipeline"), buildID)
		if err != nil {
			return fmt.Errorf("failed to get build details: %w", err)
		}

		// Print build summary
		fmt.Println("═" + strings.Repeat("═", 78) + "═")
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("  BUILD DIAGNOSIS: %s #%s", viper.GetString("pipeline"), buildID)))
		fmt.Println("═" + strings.Repeat("═", 78) + "═")
		fmt.Println()

		// Build status
		statusStyle := successStyle
		if buildInfo.Result == "FAILURE" {
			statusStyle = failureStyle
		} else if buildInfo.Result == "ABORTED" {
			statusStyle = orangeStyle
		}
		fmt.Printf("Status:   %s\n", statusStyle.Render(buildInfo.Result))
		fmt.Printf("Duration: %s\n", formatting.Duration(time.Duration(buildInfo.Duration*int(time.Millisecond))))
		fmt.Printf("URL:      %s\n", buildInfo.URL)
		fmt.Println()

		// Collect failed leaf stages with their paths
		var failedLeaves []StageWithPath
		var allLeaves []StageWithPath
		collectFailedLeafStages(job.Stages, []string{}, &failedLeaves)
		if showAllStages {
			collectAllLeafStages(job.Stages, []string{}, &allLeaves)
		}

		stagesToShow := failedLeaves
		if showAllStages {
			stagesToShow = allLeaves
		}

		if len(stagesToShow) == 0 {
			fmt.Println(successStyle.Render("✓ No failed stages found - build passed!"))
			return nil
		}

		// Show summary
		fmt.Println(infoBoldStyle.Render("FAILED STAGES:"))
		for i, item := range failedLeaves {
			fullPath := strings.Join(append(item.Path, item.Stage.Name), " > ")
			fmt.Printf("  %d. %s (Duration: %s)\n",
				i+1,
				fullPath,
				formatting.Duration(time.Duration(item.Stage.Duration*int(time.Millisecond))))
		}
		fmt.Println()

		// Show detailed logs for each stage
		fmt.Println(infoBoldStyle.Render("STAGE LOGS:"))
		fmt.Println()

		for i, item := range stagesToShow {
			printStageDividerWithPath(i+1, item.Stage, item.Path, showAllStages)

			// Get log if available
			if item.Stage.Links.Log.HREF == "" {
				fmt.Println(grayStyle.Render("  (no log available)"))
				fmt.Println()
				continue
			}

			node, err := jenkinsClient.GetStageLog(item.Stage.Links.Log.HREF)
			if err != nil {
				fmt.Println(grayStyle.Render(fmt.Sprintf("  (failed to fetch log: %v)", err)))
				fmt.Println()
				continue
			}

			// Print log (with line limit if specified)
			lines := strings.Split(node.Text, "\n")
			if maxLogLines > 0 && len(lines) > maxLogLines {
				// Show first and last lines
				half := maxLogLines / 2
				for _, line := range lines[:half] {
					fmt.Println(line)
				}
				fmt.Println(grayStyle.Render(fmt.Sprintf("\n  ... (%d lines omitted) ...\n", len(lines)-maxLogLines)))
				for _, line := range lines[len(lines)-half:] {
					fmt.Println(line)
				}
			} else {
				fmt.Println(node.Text)
			}
			fmt.Println()
		}

		// Print summary for AI
		fmt.Println("═" + strings.Repeat("═", 78) + "═")
		fmt.Println(infoBoldStyle.Render("SUMMARY FOR ANALYSIS:"))
		fmt.Printf("  Build %s had %d failed stage(s)\n", buildID, len(failedLeaves))
		if len(failedLeaves) > 0 {
			fmt.Println("  Failed stages:")
			for _, item := range failedLeaves {
				fullPath := strings.Join(append(item.Path, item.Stage.Name), " > ")
				fmt.Printf("    - %s (%s)\n", fullPath, item.Stage.Status)
			}
		}
		fmt.Println("═" + strings.Repeat("═", 78) + "═")

		return nil
	},
}

func printStageDividerWithPath(index int, stage jenkins.Stage, path []string, showStatus bool) {
	fullPath := strings.Join(append(path, stage.Name), " > ")

	statusMarker := failureStyle.Render("✗ FAILED")
	if stage.Status == "SUCCESS" {
		statusMarker = successStyle.Render("✓ SUCCESS")
	} else if stage.Status == "ABORTED" {
		statusMarker = orangeStyle.Render("⊘ ABORTED")
	}

	if showStatus {
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("─── %d. %s %s ───", index, fullPath, statusMarker)))
	} else {
		fmt.Println(infoBoldStyle.Render(fmt.Sprintf("─── %d. %s ───", index, fullPath)))
	}
	fmt.Println(grayStyle.Render(fmt.Sprintf("Stage ID: %s | Duration: %s | Node: %s",
		stage.ID,
		formatting.Duration(time.Duration(stage.Duration*int(time.Millisecond))),
		stage.ExecNode)))
	fmt.Println()
}

// collectFailedLeafStages finds only the deepest failed stages (leaf nodes with logs)
func collectFailedLeafStages(stages []jenkins.Stage, path []string, leaves *[]StageWithPath) {
	for _, stage := range stages {
		// Check if this stage has failed children
		hasFailedChildren := false
		if len(stage.StageFlowNodes) > 0 {
			for _, child := range stage.StageFlowNodes {
				if child.Status == "FAILED" || child.Status == "ABORTED" {
					hasFailedChildren = true
					break
				}
			}
		}

		// If this stage failed and either has no children or no failed children, it's a leaf
		if (stage.Status == "FAILED" || stage.Status == "ABORTED") && !hasFailedChildren {
			*leaves = append(*leaves, StageWithPath{
				Stage: stage,
				Path:  append([]string{}, path...), // Copy the path
			})
		}

		// Recursively check children
		if len(stage.StageFlowNodes) > 0 {
			newPath := append(path, stage.Name)
			collectFailedLeafStages(stage.StageFlowNodes, newPath, leaves)
		}
	}
}

// collectAllLeafStages finds all leaf stages (deepest stages with logs)
func collectAllLeafStages(stages []jenkins.Stage, path []string, leaves *[]StageWithPath) {
	for _, stage := range stages {
		// If this stage has no children, it's a leaf
		if len(stage.StageFlowNodes) == 0 {
			*leaves = append(*leaves, StageWithPath{
				Stage: stage,
				Path:  append([]string{}, path...), // Copy the path
			})
		} else {
			// Recursively check children
			newPath := append(path, stage.Name)
			collectAllLeafStages(stage.StageFlowNodes, newPath, leaves)
		}
	}
}

package cmd

import (
	"encoding/json"
	"fmt"
	"jenkins/internal/formatting"
	"jenkins/internal/jenkins"
	"strings"
	"sync"
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
		collectFailedLeafStagesForFailed(job.Stages, []string{}, &failedLeaves)

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

// stageFetchJobForFailed represents a stage to fetch with its context
type stageFetchJobForFailed struct {
	stage jenkins.Stage
	path  []string
}

// stageFetchResultForFailed represents the result of fetching a stage
type stageFetchResultForFailed struct {
	stage jenkins.Stage
	path  []string
	err   error
}

// fetchStageWorkerForFailed fetches stage details from Jenkins API
func fetchStageWorkerForFailed(jobs <-chan stageFetchJobForFailed, results chan<- stageFetchResultForFailed, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		res, err := jenkinsClient.Request("GET", job.stage.Links.Self.HREF)
		if err != nil {
			results <- stageFetchResultForFailed{err: err}
			continue
		}

		var stage jenkins.Stage
		if err := json.NewDecoder(res.Body).Decode(&stage); err != nil {
			res.Body.Close()
			results <- stageFetchResultForFailed{err: err}
			continue
		}
		res.Body.Close()

		results <- stageFetchResultForFailed{
			stage: stage,
			path:  job.path,
			err:   nil,
		}
	}
}

// collectFailedLeafStagesForFailed finds only the deepest failed stages (leaf nodes with logs)
// Uses parallel fetching with a worker pool for better performance
func collectFailedLeafStagesForFailed(stages []jenkins.Stage, path []string, leaves *[]StageWithPath) {
	const numWorkers = 10 // Number of concurrent workers

	// Create jobs and results channels
	jobs := make(chan stageFetchJobForFailed, len(stages)*2) // Buffer for all potential jobs
	results := make(chan stageFetchResultForFailed, len(stages)*2)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go fetchStageWorkerForFailed(jobs, results, &wg)
	}

	// Queue initial jobs
	for _, stage := range stages {
		jobs <- stageFetchJobForFailed{
			stage: stage,
			path:  append([]string{}, path...), // Copy the path
		}
	}

	// Start a goroutine to close results when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results and queue additional jobs for children
	processed := 0
	expectedJobs := len(stages)

	for result := range results {
		processed++

		if result.err != nil {
			verbose("Error fetching stage: %v", result.err)
			if processed >= expectedJobs {
				break
			}
			continue
		}

		stage := result.stage

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
				Path:  result.path,
			})
		}

		// Queue child stages for processing
		if len(stage.StageFlowNodes) > 0 {
			newPath := append(result.path, stage.Name)
			for _, child := range stage.StageFlowNodes {
				expectedJobs++
				jobs <- stageFetchJobForFailed{
					stage: child,
					path:  append([]string{}, newPath...), // Copy the path
				}
			}
		}

		// If we've processed all expected jobs, we're done
		if processed >= expectedJobs {
			break
		}
	}

	close(jobs)
}

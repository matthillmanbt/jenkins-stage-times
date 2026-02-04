package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"jenkins/internal/jenkins"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build [product] [branch]",
	Short: "Trigger a build on the master pipeline",
	Long: `Trigger a build on the master pipeline with specified product and branch.

Product must be one of:
  - ingredi (or rs)
  - bpam (or pra)

Branch is the TRYMAX_BRANCH to build (e.g., feature/my-branch).
Note: "origin/" will be automatically prepended if not provided.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		product := args[0]
		branch := args[1]

		// Normalize product name
		switch product {
		case "rs":
			product = "ingredi"
		case "pra":
			product = "bpam"
		case "ingredi", "bpam":
			// Already valid
		default:
			return fmt.Errorf("invalid product: %s (must be 'ingredi', 'bpam', 'rs', or 'pra')", product)
		}

		// Prepend "origin/" to branch if not already present
		if !strings.HasPrefix(branch, "origin/") {
			branch = "origin/" + branch
		}

		verbose("Triggering build for product [%s] branch [%s]", product, branch)

		params := map[string]string{
			"PRODUCT":       product,
			"TRYMAX_BRANCH": branch,
		}
		vVerbose("Build params [%#+v]", params)

		res, err := jenkinsClient.TriggerBuild(viper.GetString("pipeline"), params)
		if err != nil {
			verbose("Request error")
			return err
		}

		{
			defer res.Body.Close()
			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			verbose("Response body [%s]", string(bodyBytes))
		}

		// Get the queue location from the response
		location := res.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("no queue location in response")
		}

		fmt.Printf("Build queued successfully!\n")
		fmt.Printf("Product: %s\n", product)
		fmt.Printf("Branch:  %s\n", branch)
		fmt.Printf("Queue:   %s\n", location)
		fmt.Println()

		// Poll the queue to get the build number
		u, err := url.Parse(location)
		if err != nil {
			return err
		}
		path := u.Path[1:] + u.Fragment + u.RawQuery
		verbose("Polling queue location [%s]", path)
		p := NewURLPoller(path)
		defer p.Stop()

		for res := range p.Response {
			defer res.Body.Close()
			var queue jenkins.QueueItem
			if err := json.NewDecoder(res.Body).Decode(&queue); err != nil {
				verbose("JSON decode error trying to parse build id [%v]", err)
				break
			}

			buildNumber := strconv.Itoa(queue.Executable.Number)
			fmt.Printf("Build started: #%s\n", buildNumber)
			fmt.Printf("Monitor with: jenkins monitor %s\n", buildNumber)
			fmt.Printf("Diagnose with: jenkins diagnose %s\n", buildNumber)

			// Optionally start monitoring in background
			buildArgs := []string{"monitor", "--bg", "--pipeline", viper.GetString("pipeline"), buildNumber}
			verbose("Spawning monitor with args [%+v]", buildArgs)
			monitorCmd, err := SpawnBG(buildArgs...)
			if err != nil {
				return err
			}

			if err := monitorCmd.Wait(); err != nil {
				return err
			}

			break
		}

		return nil
	},
}

package cmd

import (
	"encoding/json"
	"fmt"
	"jenkins/internal/jenkins"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(queueCmd)
}

var queueCmd = &cobra.Command{
	Use:   "queue [queue_number]",
	Short: "Resolve a Jenkins queue number to a build number",
	Long: `Look up a Jenkins queue item by its queue number and return the associated build number.
Useful when a build was queued but the build number couldn't be determined automatically.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		queueNumber := args[0]

		path := fmt.Sprintf("queue/item/%s/api/json", queueNumber)
		verbose("Querying queue item [%s]", path)

		p := NewURLPoller(path)
		defer p.Stop()

		for res := range p.Response {
			var queue jenkins.QueueItem
			if err := json.NewDecoder(res.Body).Decode(&queue); err != nil {
				verbose("JSON decode error [%v]", err)
				res.Body.Close()
				return fmt.Errorf("failed to parse queue item %s: %w", queueNumber, err)
			}
			res.Body.Close()

			if queue.Executable.Number == 0 {
				fmt.Printf("Queue item %s is still waiting to start...\n", queueNumber)
				continue
			}

			buildNumber := strconv.Itoa(queue.Executable.Number)
			fmt.Printf("Queue item %s resolved to build #%s\n", queueNumber, buildNumber)
			fmt.Printf("Monitor with: jenkins monitor %s\n", buildNumber)
			fmt.Printf("Diagnose with: jenkins diagnose %s\n", buildNumber)
			return nil
		}

		return fmt.Errorf("queue item %s was not resolved to a build number", queueNumber)
	},
}

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"jenkins/internal/jenkins"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	tailLines    int
	headLines    int
	fetchFullLog bool
)

func init() {
	rootCmd.AddCommand(stageLogCmd)

	stageLogCmd.Flags().IntVarP(&tailLines, "tail", "t", 0, "Show only the last N lines")
	stageLogCmd.Flags().IntVarP(&headLines, "head", "n", 0, "Show only the first N lines")
	stageLogCmd.Flags().BoolVarP(&fetchFullLog, "full", "f", false, "Fetch the full log without truncation")
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
		logURL := stage.Links.Log.HREF
		if fetchFullLog {
			logURL = fmt.Sprintf("job/%s/%s/execution/node/%s/log/?consoleFull", viper.GetString("pipeline"), buildID, stageID)
		}

		var lines []string

		if !fetchFullLog {
			node, err := jenkinsClient.GetStageLog(logURL)
			if err != nil {
				return fmt.Errorf("failed to get stage log: %w", err)
			}
			// Process output based on flags
			output := node.Text
			lines = strings.Split(output, "\n")
			if tailLines > 0 && tailLines < len(lines) {
				lines = lines[len(lines)-tailLines:]
				fmt.Println(grayStyle.Render(fmt.Sprintf("(showing last %d lines)", tailLines)))
			} else if headLines > 0 && headLines < len(lines) {
				lines = lines[:headLines]
				fmt.Println(grayStyle.Render(fmt.Sprintf("(showing first %d lines)", headLines)))
			}
		} else {
			res, err := jenkinsClient.Request(http.MethodGet, logURL)
			if err != nil {
				return fmt.Errorf("failed to get full stage log: %w", err)
			}
			defer res.Body.Close()

			var outputBuilder strings.Builder
			_, err = io.Copy(&outputBuilder, res.Body)
			if err != nil {
				return fmt.Errorf("failed to read full stage log: %w", err)
			}

			text, err := extractTextFromJenkinsHTML(outputBuilder.String())
			if err != nil {
				return err
			}

			lines = strings.Split(text, "\n")
		}

		// Print the log
		for _, line := range lines {
			fmt.Println(line)
		}

		return nil
	},
}

// extractTextFromJenkinsHTML extracts plain text from Jenkins HTML console output
// Jenkins returns HTML with <pre class="console-output"> containing the log with markup
func extractTextFromJenkinsHTML(html string) (string, error) {
	// Find the <pre class="console-output"> tag
	preIdx := strings.Index(html, "console-output")
	if preIdx == -1 {
		return "", fmt.Errorf("failed to find <pre class=\"console-output\"> in response")
	}

	// Find the opening "<pre" before the class marker
	openPreIdx := strings.LastIndex(html[:preIdx], "<pre")
	if openPreIdx == -1 {
		return "", fmt.Errorf("failed to locate opening <pre> tag")
	}

	// Find the closing ">" of the opening tag
	openTagEnd := strings.Index(html[openPreIdx:], ">")
	if openTagEnd == -1 {
		return "", fmt.Errorf("failed to locate end of <pre> opening tag")
	}
	contentStart := openPreIdx + openTagEnd + 1

	// Find the closing </pre>
	closePreRel := strings.Index(html[contentStart:], "</pre>")
	if closePreRel == -1 {
		return "", fmt.Errorf("failed to locate closing </pre> tag")
	}
	contentEnd := contentStart + closePreRel

	contentHTML := html[contentStart:contentEnd]

	// Strip HTML tags (e.g., spans for ANSI coloring)
	var textBuilder strings.Builder
	inTag := false
	var tagBuilder strings.Builder

	for i := 0; i < len(contentHTML); i++ {
		c := contentHTML[i]

		if inTag {
			if c == '>' {
				inTag = false
				// Preserve line breaks for <br> tags
				tag := strings.ToLower(strings.TrimSpace(tagBuilder.String()))
				if strings.HasPrefix(tag, "br") || strings.HasPrefix(tag, "/br") {
					textBuilder.WriteByte('\n')
				}
				tagBuilder.Reset()
				continue
			}
			tagBuilder.WriteByte(c)
			continue
		}

		if c == '<' {
			inTag = true
			continue
		}

		textBuilder.WriteByte(c)
	}

	// Unescape common HTML entities
	text := textBuilder.String()
	text = strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	).Replace(text)

	return text, nil
}

// findStageByID recursively searches for a stage by ID
// Fetches full stage details via API to access nested children
func findStageByID(stages []jenkins.Stage, stageID string) *jenkins.Stage {
	for _, s := range stages {
		// Fetch full stage details to get children
		res, err := jenkinsClient.Request(http.MethodGet, s.Links.Self.HREF)
		if err != nil {
			verbose("Error fetching stage %s: %v", s.ID, err)
			continue
		}

		var stage jenkins.Stage
		if err := json.NewDecoder(res.Body).Decode(&stage); err != nil {
			res.Body.Close()
			verbose("Error decoding stage %s: %v", s.ID, err)
			continue
		}
		res.Body.Close()

		// Check if this is the stage we're looking for
		if stage.ID == stageID {
			return &stage
		}

		// Recursively search nested stages
		if len(stage.StageFlowNodes) > 0 {
			if found := findStageByID(stage.StageFlowNodes, stageID); found != nil {
				return found
			}
		}
	}
	return nil
}

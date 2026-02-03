package cmd

import (
	"errors"
	"fmt"
	"jenkins/internal/formatting"
	"jenkins/internal/jenkins"
	"jenkins/internal/util"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Goldziher/go-utils/sliceutils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type stageTime struct {
	Avg float64
	Min int
	Max int
}
type pair[T any] struct {
	Key   string
	Value T
}

const STAGE_COL_WIDTH = 60

var (
	filter  []string
	useAnd  bool
	longest bool

	jobRE = regexp.MustCompile(`^/job/[^/]+/(\d+)/`)
)

var (
	HeaderStyle  = orangeStyle.Bold(true).Align(lipgloss.Center)
	CellStyle    = textStyle.Padding(0, 1).Width(11)
	OddRowStyle  = stdRe.NewStyle().Background(gray).Inherit(CellStyle)
	EvenRowStyle = stdRe.NewStyle().Background(lipgloss.NoColor{}).Inherit(CellStyle)
	BorderStyle  = orangeStyle
)

func init() {
	rootCmd.AddCommand(timingCmd)

	timingCmd.Flags().StringArrayVarP(&filter, "filter", "f", []string{}, "Filter stage list (case insensitive)")
	timingCmd.Flags().BoolVarP(&useAnd, "and", "", false, "Combine filters with 'and' instead of 'or'")
	timingCmd.Flags().BoolVarP(&longest, "longest", "", false, "Instead of averages, print the longest matching stage")
}

var timingCmd = &cobra.Command{
	Use:   "timing",
	Short: "Summarize recent Jenkins jobs",
	Long: `Read the last 10 Jenkins jobs and summarize the
	pipeline data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jobs, err := jenkinsClient.GetJobs(viper.GetString("pipeline"))
		if err != nil {
			verbose("Request error")
			return err
		}

		var lcFilter []string

		for _, f := range filter {
			verbose("Appending filter to list [%s]", strings.ToLower(f))
			lcFilter = append(lcFilter, strings.ToLower(f))
		}

		stageMap := map[string][]jenkins.Stage{}
		successfulJobs := 0
		for _, job := range jobs {
			if job.Status != "SUCCESS" {
				verbose("Job has a status other than SUCCESS [%s][%s]", job.ID, job.Status)
				continue
			}

			successfulJobs++
			for _, stage := range job.Stages {
				if len(lcFilter) > 0 {
					var found *bool
					for _, f := range lcFilter {
						if strings.Contains(strings.ToLower(stage.Name), f) {
							vVerbose("Stage matched filter [%s][%s]", stage.Name, f)
							if !useAnd {
								found = util.Ptr(true)
								break
							} else if found == nil {
								found = util.Ptr(true)
							}
						} else {
							found = util.Ptr(false)
						}
					}
					if found == nil || !*found {
						vVerbose("Stage did not match any filter [%s][%v]", stage.Name, useAnd)
						continue
					}
				}
				stageMap[stage.Name] = append(stageMap[stage.Name], stage)
			}
		}

		verbose("Ended with [%d] stages", len(stageMap))

		if len(stageMap) == 0 {
			return errors.New(errStyle.Render("No matching, successful jobs found"))
		}

		if longest {
			printLongestStageTable(stageMap)
		} else {
			printStageTable(stageMap)
		}
		printSummary(len(stageMap), successfulJobs)

		return nil
	},
}

func printStageTable(stageMap map[string][]jenkins.Stage) {
	avgStage := []pair[stageTime]{}
	for stage, stages := range stageMap {
		durations := sliceutils.Pluck(stages, func(s jenkins.Stage) *int {
			return &s.Duration
		})
		vVerbose("Stage [%s]", stage)
		vVerbose("  %+v", durations)
		avgStage = append(avgStage, pair[stageTime]{stage, stageTime{util.Avg(durations), slices.Min(durations), slices.Max(durations)}})
	}
	sort.Slice(avgStage, func(i, j int) bool {
		return avgStage[i].Value.Avg > avgStage[j].Value.Avg
	})

	t := table.New().
		Border(lipgloss.ThickBorder()).
		BorderStyle(BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}

			switch {
			case col == 0:
				style = stdRe.NewStyle().Inline(true).Width(STAGE_COL_WIDTH).Inherit(style)
			default:
				style = stdRe.NewStyle().Align(lipgloss.Right).Inherit(style)
			}

			return style
		}).
		Headers("STAGE", "AVG", "MIN", "MAX")

	for _, p := range avgStage {
		t.Row(
			p.Key,
			formatting.Duration(time.Duration(p.Value.Avg*1000*1000)),
			formatting.Duration(time.Duration(p.Value.Min*1000*1000)),
			formatting.Duration(time.Duration(p.Value.Max*1000*1000)),
		)
	}

	fmt.Println(t)
}

func printLongestStageTable(stageMap map[string][]jenkins.Stage) {
	longestStages := []pair[jenkins.Stage]{}
	for stage, stages := range stageMap {
		longest := sliceutils.Reduce(stages[1:], func(s jenkins.Stage, c jenkins.Stage, i int, slice []jenkins.Stage) jenkins.Stage {
			vVerbose("  [%s (%d)] > [%s (%d)]", c.ID, c.Duration, s.ID, s.Duration)
			if c.Duration > s.Duration {
				vVerbose("    [%s]", c.ID)
				return c
			}
			vVerbose("    [%s]", s.ID)
			return s
		}, stages[0])
		vVerbose("Stage [%s]", stage)
		vVerbose("  %+v", longest)
		longestStages = append(longestStages, pair[jenkins.Stage]{stage, longest})
	}
	sort.Slice(longestStages, func(i, j int) bool {
		return longestStages[i].Value.Duration > longestStages[j].Value.Duration
	})

	t := table.New().
		Border(lipgloss.ThickBorder()).
		BorderStyle(BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			var style lipgloss.Style

			switch {
			case row == 0:
				return HeaderStyle
			case row%2 == 0:
				style = EvenRowStyle
			default:
				style = OddRowStyle
			}

			switch {
			case col == 0:
				style = stdRe.NewStyle().Inline(true).Width(STAGE_COL_WIDTH + 11 + 1).Inherit(style)
			default:
				style = stdRe.NewStyle().Align(lipgloss.Right).Width(11).Inherit(style)
			}

			return style
		}).
		Headers("STAGE", "TIME", "BUILD")

	for _, p := range longestStages {
		idMatch := jobRE.FindStringSubmatch(p.Value.Links.Self.HREF)
		t.Row(
			p.Key,
			formatting.Duration(time.Duration(p.Value.Duration*1000*1000)),
			idMatch[1],
		)
	}

	fmt.Println(t)
}

func printSummary(stageCount int, jobCount int) {
	style := infoBoxStyle.
		Align(lipgloss.Right).
		Width(2 + STAGE_COL_WIDTH + 3*12)

	fmt.Println(style.Render(fmt.Sprintf("Times for %d stages across %d successful jobs", stageCount, jobCount)))
}

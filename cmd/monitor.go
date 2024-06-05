package cmd

import (
	"fmt"
	"slices"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	monitorIDs []string
	doNotSpawn bool
)

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().StringArrayVarP(&monitorIDs, "build", "b", []string{}, "Build ID to monitor")
	monitorCmd.MarkFlagRequired("build")

	monitorCmd.Flags().BoolVarP(&doNotSpawn, "bg", "", false, "")
	monitorCmd.Flags().MarkHidden("bg")
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor a build and print a message when it completes of a given build ID",
	Long:  `Query Jenkins for the status of a build given the build ID until it finishes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !doNotSpawn {
			buildArgs := []string{"monitor", "--bg", "--pipeline", viper.GetString("pipeline")}
			for _, buildID := range monitorIDs {
				buildArgs = append(buildArgs, "-b", buildID)
			}
			verbose("Spawning and passing args [%+v]", buildArgs)
			cmd := SpawnBG(buildArgs...)

			return cmd.Wait()
		}

		er := make(chan error)
		bld := make(chan *WorkflowRun)
		done := make(chan bool)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		go func() {
			finished := []string{}
			for ; true; <-ticker.C {
				for _, buildID := range monitorIDs {
					if slices.Contains(finished, buildID) {
						continue
					}
					build, err := getBuildInfo(buildID)
					if err != nil {
						verbose("getBuildInfo returned error")
						er <- err
						return
					}
					vVerbose("build [%s] is building? [%v]", build.ID, build.Building)
					if !build.Building {
						bld <- build
						finished = append(finished, build.ID)
					}
				}
				vVerbose("Looping [%d] == [%d]", len(finished), len(monitorIDs))
				if len(finished) == len(monitorIDs) {
					done <- true
					return
				}
			}
		}()

		style := stdRe.NewStyle().
			Foreground(lipgloss.NoColor{})
		infoStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(orange)
		errStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(white).
			Background(red)
		successStyle := stdRe.NewStyle().
			Bold(true).
			Foreground(white).
			Background(green)

		pipeline := infoStyle.Render(viper.GetString("pipeline"))

		for {
			select {
			case build := <-bld:
				id := infoStyle.Render(build.ID)
				name := infoStyle.Render(build.DisplayName)
				rStyle := infoStyle
				if build.Result == "SUCCESS" {
					rStyle = successStyle
				} else if build.Result == "FAILURE" {
					rStyle = errStyle
				}
				result := rStyle.Render(build.Result)
				fmt.Println(style.Render(fmt.Sprintf("%s: The monitor for [%s] on branch [%s] is [%s]", id, name, pipeline, result)))
			case err := <-er:
				return err
			case <-done:
				return nil
			}
		}
	},
}

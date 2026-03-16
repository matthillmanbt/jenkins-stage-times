package cmd

import (
	"fmt"
	"jenkins/internal/jenkins"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// MonitorPollInterval is how often to check build status
	MonitorPollInterval = 5 * time.Second
)

var (
	doNotSpawn bool
)

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().BoolVarP(&doNotSpawn, "bg", "", false, "")
	monitorCmd.Flags().MarkHidden("bg")
}

var monitorCmd = &cobra.Command{
	Use:   "monitor [build_id] [...build_id]",
	Short: "Monitor a build and print a message when it completes of a given build IDs",
	Long:  `Query Jenkins for the status of a build given the build ID until it finishes`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !doNotSpawn {
			pipeline := viper.GetString("pipeline")
			fmt.Printf("Monitoring build(s) %s on pipeline [%s]...\n", strings.Join(args, ", "), pipeline)
			buildArgs := []string{"monitor", "--bg", "--pipeline", pipeline}
			buildArgs = append(buildArgs, args...)
			verbose("Spawning and passing args [%+v]", buildArgs)
			cmd, err := SpawnBG(buildArgs...)
			if err != nil {
				return err
			}

			return cmd.Wait()
		}

		er := make(chan error)
		bld := make(chan *jenkins.WorkflowRun)
		done := make(chan bool)
		ticker := time.NewTicker(MonitorPollInterval)
		defer ticker.Stop()
		go func() {
			finished := []string{}
			for ; true; <-ticker.C {
				for _, buildID := range args {
					if slices.Contains(finished, buildID) {
						continue
					}
					build, err := jenkinsClient.GetBuildInfo(viper.GetString("pipeline"), buildID)
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
				vVerbose("Looping [%d] == [%d]", len(finished), len(args))
				if len(finished) == len(args) {
					done <- true
					return
				}
			}
		}()

		pipeline := infoBoldStyle.Render(viper.GetString("pipeline"))

		for {
			select {
			case build := <-bld:
				id := infoBoldStyle.Render(build.ID)
				name := infoBoldStyle.Render(build.DisplayName)
				rStyle := infoBoldStyle
				if build.Result == "SUCCESS" {
					rStyle = successStyle
				} else if build.Result == "FAILURE" {
					rStyle = failureStyle
				}
				result := rStyle.Render(build.Result)
				fmt.Println(noStyle.Render(fmt.Sprintf("%s: The monitor for [%s] on branch [%s] is [%s]", id, name, pipeline, result)))
			case err := <-er:
				return err
			case <-done:
				fmt.Println(noStyle.Render(fmt.Sprintf("Done monitoring %d build(s) on pipeline [%s].", len(args), pipeline)))
				return nil
			}
		}
	},
}

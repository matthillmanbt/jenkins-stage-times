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
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push [build_id] [subdomain]",
	Short: "Run the build-site pipeline and push a build to a site.",
	Long:  `Push the given build for the specified pipeline to [subdomain].dev.bomgar.com`,
	Args:  cobra.MatchAll(cobra.ExactArgs(2), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		buildNumber := strings.ToLower(args[0])

		if buildNumber == "rs" || buildNumber == "pra" {
			searchProduct := viper.GetString("products.rs.search_name")
			if buildNumber == "pra" {
				searchProduct = viper.GetString("products.pra.search_name")
			}
			latestBuild, err := jenkinsClient.GetLatestBuild(viper.GetString("pipeline"), searchProduct, "origin/master")
			if err != nil {
				return err
			}
			buildNumber = latestBuild.ID
		}

		domain := viper.GetString("deployment.domain")
		verbose("Pushing [%s] to [%s.%s]", buildNumber, args[1], domain)

		query := map[string]string{
			"PROJECT_NAME": viper.GetString("pipeline"),
			"BUILD_NUMBER": buildNumber,
			"SUBDOMAIN":    args[1],
		}
		vVerbose("build-site params [%#+v]", query)
		res, err := jenkinsClient.TriggerBuild("build-site", query)
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
			verbose("Body [%s]", string(bodyBytes))
		}

		fmt.Printf("Push queued successfully!\n")
		fmt.Printf("Build:     %s\n", buildNumber)
		fmt.Printf("Target:    %s.%s\n", args[1], domain)
		fmt.Println()

		location := res.Header.Get("Location")
		u, err := url.Parse(location)
		if err != nil {
			return err
		}
		path := u.Path[1:] + u.Fragment + u.RawQuery
		queueNumber := QueueNumberFromPath(location)
		verbose("Polling location [%s][%s] (queue #%s)", location, path, queueNumber)
		p := NewURLPoller(path)
		defer p.Stop()

		for res := range p.Response {
			var queue jenkins.QueueItem
			if err := json.NewDecoder(res.Body).Decode(&queue); err != nil {
				verbose("JSON decode error trying to parse build id [%v]", err)
				res.Body.Close()
				fmt.Printf("Could not resolve queue item %s to a build number.\n", queueNumber)
				fmt.Printf("Resolve manually with: jenkins queue %s\n", queueNumber)
				return nil
			}
			res.Body.Close()

			if queue.Executable.Number == 0 {
				verbose("Build not yet started, still queued...")
				continue
			}

			buildSiteNumber := strconv.Itoa(queue.Executable.Number)
			fmt.Printf("build-site started: #%s\n", buildSiteNumber)
			fmt.Printf("Monitor with: jenkins monitor --pipeline build-site %s\n", buildSiteNumber)

			buildArgs := []string{"monitor", "--bg", "--pipeline", "build-site"}
			buildArgs = append(buildArgs, buildSiteNumber)
			verbose("Spawning and passing args [%+v]", buildArgs)
			cmd, err := SpawnBG(buildArgs...)
			if err != nil {
				return err
			}

			if err := cmd.Wait(); err != nil {
				return err
			}

			break
		}

		return nil
	},
}

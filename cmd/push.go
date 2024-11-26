package cmd

import (
	"encoding/json"
	"io"
	"net/http"
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
			searchProduct := "ingredi"
			if buildNumber == "pra" {
				searchProduct = "bpam"
			}
			latestBuild, err := getLatestBuild(searchProduct, "origin/master")
			if err != nil {
				return err
			}
			buildNumber = latestBuild.ID
		}

		verbose("Pushing [%s] to [%s.dev.bomgar.com]", buildNumber, args[1])

		query := map[string]string{
			"PROJECT_NAME": viper.GetString("pipeline"),
			"BUILD_NUMBER": buildNumber,
			"SUBDOMAIN":    args[1],
		}
		vVerbose("build-site params [%#+v]", query)
		queryUrl := "job/build-site/buildWithParameters"
		res, err := jenkinsRequestWithMethod(http.MethodPost, queryUrl, query)
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

		location := res.Header.Get("Location")
		u, err := url.Parse(location)
		if err != nil {
			return err
		}
		path := u.Path[1:] + u.Fragment + u.RawQuery
		verbose("Polling location [%s][%s]", location, path)
		p := NewURLPoller(path)

		for res := range p.Response {
			defer res.Body.Close()
			var queue QueueItem
			if err := json.NewDecoder(res.Body).Decode(&queue); err != nil {
				verbose("JSON decode error trying to parse build id [%v]", err)
				break
			}

			buildArgs := []string{"monitor", "--bg", "--pipeline", viper.GetString("pipeline")}
			buildArgs = append(buildArgs, strconv.Itoa(queue.Executable.Number))
			verbose("Spawning and passing args [%+v]", buildArgs)
			cmd := SpawnBG(buildArgs...)

			if err := cmd.Wait(); err != nil {
				return err
			}
		}

		return nil
	},
}

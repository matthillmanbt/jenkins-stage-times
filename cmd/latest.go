package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	isRS   bool
	isPRA  bool
	branch string

	searchProduct  = "ingredi"
	displayProduct = "RS"
)

func init() {
	rootCmd.AddCommand(latestCmd)

	latestCmd.Flags().StringVarP(&branch, "branch", "b", "master", "branch")

	latestCmd.Flags().BoolVarP(&isRS, "rs", "r", false, "Remote Support")
	latestCmd.Flags().BoolVarP(&isPRA, "pra", "p", false, "PRA")
	latestCmd.MarkFlagsOneRequired("rs", "pra")
	latestCmd.MarkFlagsMutuallyExclusive("rs", "pra")
}

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Print the latest build for a product",
	Long:  `Query Jenkins for the latest build of a branch for a product and print the result`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if isPRA {
			searchProduct = "bpam"
			displayProduct = "PRA"
		}

		if !strings.HasPrefix(branch, "origin/") {
			branch = fmt.Sprintf("origin/%s", branch)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("job/%s/api/json", viper.Get("pipeline"))
		query := map[string]string{"tree": "builds[id,fullDisplayName,actions[parameters[name,value]]]"}
		res, err := jenkinsRequest(url, query)
		if err != nil {
			verbose("Request error")
			return err
		}
		defer res.Body.Close()

		var job WorkflowJob
		if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
			verbose("JSON decode error")
			return err
		}

		var latestBuild *WorkflowRun
		for _, run := range job.Builds {
			vVerbose("Build %s", run.ID)
			for _, action := range run.Actions {
				if len(action.Parameters) > 0 {
					bIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool { return p.Name == "TRYMAX_BRANCH" })
					pIdx := slices.IndexFunc(action.Parameters, func(p WorkflowParameter) bool { return p.Name == "PRODUCT" })
					vVerbose("  %d = %v", bIdx, action.Parameters[bIdx].Value)
					vVerbose("  %d = %v", pIdx, action.Parameters[pIdx].Value)
					if action.Parameters[pIdx].Value == searchProduct && action.Parameters[bIdx].Value == branch {
						latestBuild = &run
						break
					}
				}
			}

			if latestBuild != nil {
				break
			}
		}

		if latestBuild == nil {
			fmt.Println(errStyle.Render("No builds found for %s", displayProduct))
			os.Exit(1)
		}

		style := stdRe.NewStyle().
			Bold(true).
			Foreground(white).
			Background(orange).
			Padding(1, 6)

		fmt.Println(style.Render(fmt.Sprintf("Latest build for %s on branch [%s] is %s", displayProduct, branch, latestBuild.ID)))

		return nil
	},
}

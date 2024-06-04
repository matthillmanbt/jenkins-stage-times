package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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
		latestBuild, err := getLatestBuild(searchProduct, branch)

		if err != nil {
			verbose("latestBuild returned error")
			return err
		}

		if latestBuild == nil {
			return errors.New(errStyle.Render("No builds found for %s", displayProduct))
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

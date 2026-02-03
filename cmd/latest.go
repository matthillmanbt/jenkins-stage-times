package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	isRS    bool
	isPRA   bool
	onlyNum bool
	branch  string

	searchProduct  string
	displayProduct string
)

func init() {
	rootCmd.AddCommand(latestCmd)

	latestCmd.Flags().StringVarP(&branch, "branch", "b", "master", "branch")

	latestCmd.Flags().BoolVarP(&onlyNum, "", "n", false, "Only echo the build number")
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
			searchProduct = viper.GetString("products.pra.search_name")
			displayProduct = viper.GetString("products.pra.display_name")
		} else {
			searchProduct = viper.GetString("products.rs.search_name")
			displayProduct = viper.GetString("products.rs.display_name")
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

		if onlyNum {
			fmt.Println(latestBuild.ID)
		} else {
			fmt.Println(infoBoxStyle.Render(fmt.Sprintf("Latest build for %s on branch [%s] is %s", displayProduct, branch, latestBuild.ID)))
		}

		return nil
	},
}

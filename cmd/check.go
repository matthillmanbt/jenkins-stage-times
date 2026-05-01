package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify Jenkins API connectivity",
	Long: `Test connectivity to Jenkins and validate that your environment is properly
configured. This command makes a simple API call to verify credentials and
connectivity to the Jenkins instance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose("Checking Jenkins connectivity...")

		version, err := jenkinsClient.GetSystemInfo()
		if err != nil {
			return fmt.Errorf("connectivity check failed: %w", err)
		}

		// Format and display results
		fmt.Println("✓ Successfully connected to Jenkins")
		fmt.Println()

		hostStyle := infoBoldStyle.Render("Host:")
		fmt.Printf("%s %s\n", hostStyle, viper.GetString("host"))

		userStyle := infoBoldStyle.Render("User:")
		fmt.Printf("%s %s\n", userStyle, viper.GetString("user"))

		versionStyle := infoBoldStyle.Render("Version:")
		fmt.Printf("%s %s\n", versionStyle, version)

		return nil
	},
}

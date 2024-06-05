package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	host     string
	user     string
	key      string
	pipeline string
	Verbose  int
)

var rootCmd = &cobra.Command{
	Use:   "jenkins",
	Short: "Summarize recent Jenkins jobs",
	Long: `Read the last 10 Jenkins jobs and summarize the
	pipeline data.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var (
			vHost = viper.Get("host")
			vUser = viper.Get("user")
			vKey  = viper.Get("key")
		)
		if vHost == nil || vHost == "" {
			return fmt.Errorf("you must provide a host")
		}
		if vUser == nil || vKey == nil || vUser == "" || vKey == "" {
			return fmt.Errorf("you must provide both a username and an API key")
		}

		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	viper.SetEnvPrefix("jenkins")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jenkins.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	rootCmd.PersistentFlags().StringVar(&host, "host", "", "Jenkins host")
	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "Jenkins host")
	viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))
	rootCmd.PersistentFlags().StringVar(&key, "key", "", "Jenkins host")
	viper.BindPFlag("key", rootCmd.PersistentFlags().Lookup("key"))
	rootCmd.PersistentFlags().StringVar(&pipeline, "pipeline", "master", "Jenkins pipeline to analyze")
	viper.BindPFlag("pipeline", rootCmd.PersistentFlags().Lookup("pipeline"))
	viper.SetDefault("pipeline", "master")

	rootCmd.PersistentFlags().CountVarP(&Verbose, "verbose", "v", "verbose output")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".jenkins")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func flagsContain(flags []string, contains ...string) bool {
	for _, flag := range contains {
		if slices.Contains(flags, flag) {
			return true
		}
	}
	return false
}

func setDefaultCommandIfNonePresent(defaultCommand string) {
	// Taken from cobra source code in command.go::ExecuteC()
	var cmd *cobra.Command
	var err error
	var flags []string
	if rootCmd.TraverseChildren {
		cmd, flags, err = rootCmd.Traverse(os.Args[1:])
	} else {
		cmd, flags, err = rootCmd.Find(os.Args[1:])
	}

	// If no command was on the CLI, then cmd will return with
	// the value of rootCmd.Use (which would run the help output
	// in the full Execute() command)
	if err != nil || cmd.Use == rootCmd.Use {
		if !flagsContain(flags, "-v", "-h", "--version", "--help") {
			rootCmd.SetArgs(append(os.Args[1:], defaultCommand))
		}
	}
}

func Execute() {
	setDefaultCommandIfNonePresent("timing")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

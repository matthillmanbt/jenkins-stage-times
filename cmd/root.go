package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	host     string
	user     string
	key      string
	pipeline string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jenkins",
	Short: "Summarize recent Jenkins jobs",
	Long: `Read the last 10 Jenkins jobs and summarize the
	pipeline data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{}
		apiKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", viper.Get("user"), viper.Get("key"))))

		res, err := doRequest(client, fmt.Sprintf("%s/job/%s/wfapi/runs", viper.Get("host"), viper.Get("pipeline")), apiKey)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		var jobs []Job
		if err := json.NewDecoder(res.Body).Decode(&jobs); err != nil {
			return err
		}

		stageMap := map[string][]int{}
		successfulJobs := 0
		for _, job := range jobs {
			if job.Status != "SUCCESS" {
				continue
			}
			successfulJobs++
			for _, stage := range job.Stages {
				stageMap[stage.Name] = append(stageMap[stage.Name], stage.Duration)
			}
		}

		type pair struct {
			Key   string
			Value float64
		}

		avgStage := []pair{}
		for stage, durations := range stageMap {
			avgStage = append(avgStage, pair{stage, avg(durations)})
		}

		sort.Slice(avgStage, func(i, j int) bool {
			return avgStage[i].Value > avgStage[j].Value
		})

		re := lipgloss.NewRenderer(os.Stdout)

		var (
			orange = lipgloss.Color("#FF5500")
			gray   = lipgloss.Color("245")
			white  = lipgloss.Color("#FFFFFF")
			// HeaderStyle is the lipgloss style used for the table headers.
			HeaderStyle = re.NewStyle().Foreground(orange).Bold(true).Align(lipgloss.Center)
			// CellStyle is the base lipgloss style used for the table rows.
			CellStyle = re.NewStyle().Padding(0, 1).Width(50)
			// OddRowStyle is the lipgloss style used for odd-numbered table rows.
			OddRowStyle = CellStyle.Foreground(gray)
			// EvenRowStyle is the lipgloss style used for even-numbered table rows.
			EvenRowStyle = CellStyle.Foreground(white)
			// BorderStyle is the lipgloss style used for the table border.
			BorderStyle = lipgloss.NewStyle().Foreground(orange)
		)

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

				return style
			}).
			Headers("STAGE", "TIME")

		for _, p := range avgStage {
			t.Row(p.Key, fmtDuration(time.Duration(p.Value*1000*1000)))
		}

		fmt.Println(t)

		style := lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF5500")).
			Padding(1, 6).
			Width(102)

		fmt.Println(style.Render(fmt.Sprintf("Averages for %d stages across %d successful jobs", len(avgStage), successfulJobs)))

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	viper.SetEnvPrefix("jenkins")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jenkins.yaml)")
	viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))
	rootCmd.PersistentFlags().StringVar(&host, "host", "", "Jenkins host")
	viper.BindPFlag("host", rootCmd.Flags().Lookup("host"))
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "Jenkins host")
	viper.BindPFlag("user", rootCmd.Flags().Lookup("user"))
	rootCmd.PersistentFlags().StringVar(&key, "key", "", "Jenkins host")
	viper.BindPFlag("key", rootCmd.Flags().Lookup("key"))
	rootCmd.PersistentFlags().StringVar(&pipeline, "pipeline", "master", "Jenkins pipeline to analyze")
	viper.BindPFlag("pipeline", rootCmd.Flags().Lookup("pipeline"))
	viper.SetDefault("pipeline", "master")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".jenkins" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".jenkins")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

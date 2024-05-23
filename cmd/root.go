package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
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
	filter   string
)

var rootCmd = &cobra.Command{
	Use:   "jenkins",
	Short: "Summarize recent Jenkins jobs",
	Long: `Read the last 10 Jenkins jobs and summarize the
	pipeline data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			host = viper.Get("host")
			user = viper.Get("user")
			key  = viper.Get("key")
		)

		if host == "" {
			return fmt.Errorf("you must provide a host")
		}
		if user == "" || key == "" {
			return fmt.Errorf("you must provide both a username and an API key")
		}

		client := &http.Client{}
		apiKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, key)))

		res, err := doRequest(client, fmt.Sprintf("%s/job/%s/wfapi/runs", host, viper.Get("pipeline")), apiKey)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		var (
			jobs     []Job
			lcFilter = strings.ToLower(filter)
		)
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
				if lcFilter != "" && !strings.Contains(strings.ToLower(stage.Name), lcFilter) {
					continue
				}
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

		if len(avgStage) == 0 {
			errRe := lipgloss.NewRenderer(os.Stderr)
			style := errRe.NewStyle().
				Bold(true).
				Align(lipgloss.Center).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#FF0000")).
				Padding(1, 6).
				Width(102)

			fmt.Println(style.Render("No matching, successful jobs found"))

			os.Exit(1)
		}

		sort.Slice(avgStage, func(i, j int) bool {
			return avgStage[i].Value > avgStage[j].Value
		})

		re := lipgloss.NewRenderer(os.Stdout)

		var (
			orange       = lipgloss.Color("#FF5500")
			gray         = lipgloss.Color("245")
			white        = lipgloss.Color("#FFFFFF")
			HeaderStyle  = re.NewStyle().Foreground(orange).Bold(true).Align(lipgloss.Center)
			CellStyle    = re.NewStyle().Padding(0, 1).Width(50)
			OddRowStyle  = CellStyle.Foreground(gray)
			EvenRowStyle = CellStyle.Foreground(white)
			BorderStyle  = lipgloss.NewStyle().Foreground(orange)
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

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

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

	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter stage list (case insensitive)")
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

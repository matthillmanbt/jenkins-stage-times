package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
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
	filter   []string
	Verbose  int

	orange = lipgloss.Color("#FF5500")
	gray   = lipgloss.Color("#222222")
	white  = lipgloss.Color("#FFFFFF")
	red    = lipgloss.Color("#FF0000")
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
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("job/%s/wfapi/runs", viper.Get("pipeline"))
		res, err := jenkinsRequest(url)
		if err != nil {
			verbose("Request error")
			return err
		}
		defer res.Body.Close()

		var (
			jobs     []Job
			lcFilter []string
		)

		if err := json.NewDecoder(res.Body).Decode(&jobs); err != nil {
			verbose("JSON decode error")
			return err
		}

		for _, f := range filter {
			verbose("Appending filter to list [%s]", strings.ToLower(f))
			lcFilter = append(lcFilter, strings.ToLower(f))
		}

		stageMap := map[string][]int{}
		successfulJobs := 0
		for _, job := range jobs {
			if job.Status != "SUCCESS" {
				verbose("Job has a status other than SUCCESS [%s][%s]", job.ID, job.Status)
				continue
			}

			successfulJobs++
			for _, stage := range job.Stages {
				if len(lcFilter) > 0 {
					found := false
					for _, f := range lcFilter {
						if strings.Contains(strings.ToLower(stage.Name), f) {
							vVerbose("Stage matched filter [%s][%s]", stage.Name, f)
							found = true
							break
						}
					}
					if !found {
						vVerbose("Stage did not match any filter [%s]", stage.Name)
						continue
					}
				}
				stageMap[stage.Name] = append(stageMap[stage.Name], stage.Duration)
			}
		}

		type stageTime struct {
			Avg float64
			Min int
			Max int
		}
		type pair struct {
			Key   string
			Value stageTime
		}

		avgStage := []pair{}
		for stage, durations := range stageMap {
			vVerbose("Stage [%s]", stage)
			vVerbose("  %+v", durations)
			avgStage = append(avgStage, pair{stage, stageTime{avg(durations), slices.Min(durations), slices.Max(durations)}})
		}

		verbose("Ended with [%d] stages to print", len(avgStage))

		if len(avgStage) == 0 {
			errRe := lipgloss.NewRenderer(os.Stderr)
			style := errRe.NewStyle().
				Bold(true).
				Align(lipgloss.Center).
				Foreground(white).
				Background(red).
				Padding(1, 6).
				Width(102)

			fmt.Println(style.Render("No matching, successful jobs found"))

			os.Exit(1)
		}

		sort.Slice(avgStage, func(i, j int) bool {
			return avgStage[i].Value.Avg > avgStage[j].Value.Avg
		})

		re := lipgloss.NewRenderer(os.Stdout)

		var (
			HeaderStyle  = re.NewStyle().Foreground(orange).Bold(true).Align(lipgloss.Center)
			CellStyle    = re.NewStyle().Padding(0, 1).Width(11).Foreground(white)
			OddRowStyle  = re.NewStyle().Background(gray).Inherit(CellStyle)
			EvenRowStyle = re.NewStyle().Background(lipgloss.NoColor{}).Inherit(CellStyle)
			BorderStyle  = re.NewStyle().Foreground(orange)
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

				switch {
				case col == 0:
					style = re.NewStyle().Width(50).Inherit(style)
				default:
					style = re.NewStyle().Align(lipgloss.Right).Inherit(style)
				}

				return style
			}).
			Headers("STAGE", "AVG", "MIN", "MAX")

		for _, p := range avgStage {
			t.Row(
				p.Key,
				fmtDuration(time.Duration(p.Value.Avg*1000*1000)),
				fmtDuration(time.Duration(p.Value.Min*1000*1000)),
				fmtDuration(time.Duration(p.Value.Max*1000*1000)),
			)
		}

		fmt.Println(t)

		style := lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Right).
			Foreground(white).
			Background(orange).
			Padding(1, 6).
			Width(2 + 50 + 3*12)

		fmt.Println(style.Render(fmt.Sprintf("Times for %d stages across %d successful jobs", len(avgStage), successfulJobs)))

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

	rootCmd.PersistentFlags().CountVarP(&Verbose, "verbose", "v", "verbose output")

	rootCmd.Flags().StringArrayVarP(&filter, "filter", "f", []string{}, "Filter stage list (case insensitive)")
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

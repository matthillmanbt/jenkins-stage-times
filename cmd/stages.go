package cmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(orange)

type sortColumn int64

const (
	name sortColumn = iota
	status
	duration
	none
)

var sortKeyMap = map[string]sortColumn{
	"n": name,
	"s": status,
	"d": duration,
}

var columns = []table.Column{
	{Title: "Name", Width: 50},
	{Title: "Status", Width: 10},
	{Title: "Duration", Width: 10},
	{Title: "ID", Width: 10},
}

type model struct {
	jobs []Job

	job   *Job
	stage *Stage
	node  *Node

	table table.Model
	sort  sortColumn
	asc   bool

	viewport viewport.Model
	ready    bool
	size     tea.WindowSizeMsg

	filter     textinput.Model
	showFilter bool
	priorVal   string
}

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1).Foreground(textColor).BorderForeground(orange)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()

	headStyle = infoBoxStyle.Align(lipgloss.Center).Width(90)
	helpStyle = grayStyle.Align(lipgloss.Center).Width(90)
)

func init() {
	rootCmd.AddCommand(stagesCmd)
}

var stagesCmd = &cobra.Command{
	Use:   "stages [build_id?]",
	Short: "Show stage info for a build, or list all recent builds",
	Long:  `Given a build ID, show all the pipeline steps for browsing and digging into logs for individual stages. If no build ID is given, a list of recent jobs will be shown.`,
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("job/%s/wfapi/runs", viper.Get("pipeline"))
		res, err := jenkinsRequest(url)
		if err != nil {
			verbose("Request error")
			return err
		}
		defer res.Body.Close()
		var jobs []Job

		if err := json.NewDecoder(res.Body).Decode(&jobs); err != nil {
			verbose("JSON decode error")
			return err
		}

		var job *Job
		if len(args) > 0 {
			url := fmt.Sprintf("job/%s/%s/wfapi/describe", viper.Get("pipeline"), args[0])
			res, err := jenkinsRequest(url)
			if err != nil {
				verbose("Request error")
				return err
			}
			defer res.Body.Close()

			if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
				verbose("JSON decode error")
				return err
			}
		}

		km := table.DefaultKeyMap()
		km.LineUp.SetKeys("up", "j")
		km.LineDown.SetKeys("down", "k")
		km.PageUp.SetKeys("pgup")
		km.PageDown.SetKeys("pgdown")
		km.HalfPageUp.SetKeys("ctrl+u")
		km.HalfPageDown.SetKeys("ctrl+d")

		t := table.New(
			table.WithColumns(columns),
			table.WithRows([]table.Row{}),
			table.WithFocused(true),
			table.WithHeight(25),
			table.WithKeyMap(km),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(orange).
			BorderBottom(true).
			Foreground(textColor).
			Bold(true)
		s.Cell = s.Cell.Foreground(textColor)
		s.Selected = s.Selected.
			Foreground(textColor).
			Background(cyan).
			Bold(false)
		t.SetStyles(s)

		filter := textinput.New()
		filter.Placeholder = "Type to filter stage list"
		filter.CharLimit = 100
		filter.PromptStyle = orangeStyle
		filter.TextStyle = orangeStyle
		filter.PlaceholderStyle = grayStyle
		filter.Cursor.Style = orangeStyle

		p := tea.NewProgram(model{jobs: jobs, job: job, table: t, sort: none, filter: filter})
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	vVerbose("Update() [%+#v]", msg)
	var stages []Stage
	if m.stage != nil {
		stages = m.stage.StageFlowNodes
	} else if m.job != nil {
		stages = m.job.Stages
	}

	var cmd tea.Cmd
	showing := false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.node == nil {
			if m.showFilter {
				switch msg.String() {
				case "enter":
					m.filter.Blur()
					m.showFilter = false
				case "esc":
					m.filter.Blur()
					m.showFilter = false
					m.filter.SetValue(m.priorVal)
				}
			} else {
				switch msg.String() {
				case "q":
					return m, tea.Quit

				case "n", "d", "s":
					col := sortKeyMap[msg.String()]
					if m.sort == col {
						if !m.asc {
							m.sort = none
						} else {
							m.asc = false
						}
					} else {
						m.sort = col
						m.asc = true
					}

				case "f":
					if m.job != nil {
						m.showFilter = true
						showing = true
						m.filter.Focus()
						m.priorVal = m.filter.Value()
					}

				case "c":
					m.filter.SetValue("")

				case "esc", "left":
					if m.stage != nil {
						m.stage = nil
						stages = m.job.Stages
					} else {
						m.job = nil
					}

				case "enter", "right":
					id := m.table.SelectedRow()[3]
					if m.job == nil {
						// We're showing a list of jobs
						jIdx := slices.IndexFunc(m.jobs, func(p Job) bool { return p.ID == id })
						return m, getJobInfo(m.jobs[jIdx])
					}
					// We're showing a list of stages for a given job
					sIdx := slices.IndexFunc(stages, func(p Stage) bool { return p.ID == id })
					return m, getStageInfo(stages[sIdx])
				}
			}
		} else {
			switch msg.String() {
			case "g":
				m.viewport.GotoTop()
			case "G":
				m.viewport.GotoBottom()

			case "q", "esc":
				m.node = nil
				m.ready = false
				return m, tea.ExitAltScreen
			}
		}
	case Stage:
		vVerbose("MSG Stage [%+#v]", msg)
		m.stage = &msg
		stages = m.stage.StageFlowNodes

	case Node:
		vVerbose("MSG Node")
		m.node = &msg
		m.updateViewport()
		return m, tea.EnterAltScreen

	case Job:
		vVerbose("MSG Job")
		m.job = &msg
		stages = m.job.Stages

	case tea.WindowSizeMsg:
		vVerbose("MSG WindowSizeMsg")
		m.size = msg
		m.updateViewport()
	}

	if m.node != nil {
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	if m.job != nil {
		m.sortTableStages(stages)
	} else {
		m.sortTableJobs(m.jobs)
	}

	if m.showFilter {
		if !showing {
			m.table.GotoTop()
			m.filter, cmd = m.filter.Update(msg)
		} else {
			cmd = m.filter.Cursor.BlinkCmd()
		}
	} else {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	vVerbose("View()")
	if m.node != nil {
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	}
	var s string
	if m.job != nil {
		s += headStyle.Render(fmt.Sprintf("Stages for: %s", m.job.Name)) + "\n"
	} else {
		s += headStyle.Render("Recent Jobs") + "\n"
	}
	s += baseStyle.Render(m.table.View()) + "\n"
	if m.showFilter {
		s += m.filter.View()
	} else {
		if m.job != nil && m.stage == nil && len(m.filter.Value()) > 0 {
			s += grayStyle.Render(fmt.Sprintf("Filter: %s\n", m.filter.Value()))
		}
		helpString := "n/s/d: sort by column"
		if m.job != nil && m.stage == nil {
			helpString += " f: filter"
			if len(m.filter.Value()) > 0 {
				helpString += " c: clear"
			}
		}
		helpString += " ⏎/→: details"
		if m.job != nil || m.stage != nil {
			helpString += " ⎋/←: back"
		}
		helpString += " ⌃+c/q: quit"
		s += helpStyle.Render(helpString) + "\n"
	}
	return s
}

func getJobInfo(job Job) tea.Cmd {
	vVerbose("getJobInfo()")
	return func() tea.Msg {
		url := fmt.Sprintf("job/%s/%s/wfapi/describe", viper.Get("pipeline"), job.ID)
		res, err := jenkinsRequest(url)
		if err != nil {
			verbose("Request error")
			return err
		}
		defer res.Body.Close()

		var job Job
		if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
			verbose("JSON decode error")
			return err
		}

		return job
	}
}

func getStageInfo(stage Stage) tea.Cmd {
	vVerbose("getStageInfo()")
	return func() tea.Msg {
		vVerbose("getStageInfo() MSG")
		if stage.Links.Log.HREF == "" {
			vVerbose("  -> no Log HREF")
			res, err := jenkinsRequest(stage.Links.Self.HREF)
			if err != nil {
				verbose("Request error")
				return nil
			}
			defer res.Body.Close()

			var stg Stage
			if err := json.NewDecoder(res.Body).Decode(&stg); err != nil {
				verbose("JSON decode error")
				return err
			}

			if len(stg.StageFlowNodes) > 0 {
				vVerbose("  -> returning stage")
				return stg
			} else if stage.Links.Log.HREF != "" {
				vVerbose("  -> returning getStageInfo")
				return getStageInfo(stg)
			} else {
				vVerbose("  -> returning nil")
				return nil
			}
		}

		vVerbose("  -> no Log HREF")
		// https://jenkins.bt3ng.com/job/master/21011/execution/node/967/log/?consoleFull
		// https://jenkins.bt3ng.com/job/master/21011/execution/node/967/wfapi/log
		// strings.ReplaceAll(stage.Links.Log.HREF, "/wfapi/log", "/log/?consoleFull")
		res, err := jenkinsRequest(stage.Links.Log.HREF)
		if err != nil {
			verbose("Request error")
			return nil
		}
		defer res.Body.Close()

		var node Node
		if err := json.NewDecoder(res.Body).Decode(&node); err != nil {
			verbose("JSON decode error")
			return err
		}

		vVerbose("  -> returning node")
		return node
	}
}

func (m model) headerView() string {
	var title string
	if m.stage != nil {
		title = titleStyle.Render(m.stage.Name)
	} else {
		title = titleStyle.Render("Console Output")
	}
	line := orangeStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := orangeStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *model) updateViewport() {
	vVerbose("updateViewport() %v %v", m.node != nil, m.ready)
	if m.node != nil {
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.ready {
			m.viewport = viewport.New(m.size.Width, m.size.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = false
			vVerbose("setting text with length %d", len(m.node.Text))
			m.viewport.SetContent(m.node.Text)
			vVerbose("line count is now %d", m.viewport.TotalLineCount())
			m.ready = true
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = m.size.Width
			m.viewport.Height = m.size.Height - verticalMarginHeight
		}
	}
}

func (m *model) sortTableJobs(jobs []Job) {
	ls := make([]Base, len(jobs))
	for i, v := range jobs {
		ls[i].Links = v.Links
		ls[i].ID = v.ID
		ls[i].Name = v.Name
		ls[i].Status = v.Status
		ls[i].StartTime = v.StartTime
		ls[i].Duration = v.Duration
	}

	m.sortTable(ls)
}

func (m *model) sortTableStages(stages []Stage) {
	ls := make([]Base, len(stages))
	for i, v := range stages {
		ls[i].Links = v.Links
		ls[i].ID = v.ID
		ls[i].Name = v.Name
		ls[i].Status = v.Status
		ls[i].StartTime = v.StartTime
		ls[i].Duration = v.Duration
	}

	m.sortTable(ls)
}

func (m *model) sortTable(stages []Base) {

	rows := []table.Row{}

	sort.Slice(stages, func(i, j int) bool {
		switch {
		case m.sort == name && m.asc:
			return stages[i].Name < stages[j].Name
		case m.sort == name:
			return stages[i].Name > stages[j].Name
		case m.sort == status && m.asc:
			return stages[i].Status < stages[j].Status
		case m.sort == status:
			return stages[i].Status > stages[j].Status
		case m.sort == duration && m.asc:
			return stages[i].Duration < stages[j].Duration
		case m.sort == duration:
			return stages[i].Duration > stages[j].Duration
		case m.sort == none && m.asc:
			return stages[i].StartTime.Time.Before(stages[j].StartTime.Time)
		}
		return stages[i].StartTime.Time.After(stages[j].StartTime.Time)
	})

	lcFilter := strings.ToLower(m.filter.Value())
	for _, s := range stages {
		if m.job != nil && m.stage == nil && len(lcFilter) > 0 {
			if strings.Contains(strings.ToLower(s.Name), lcFilter) ||
				strings.Contains(strings.ToLower(s.Status), lcFilter) {
				vVerbose("Stage matched filter [%s][%s]:[%s]", s.Name, s.Status, lcFilter)
			} else {
				continue
			}
		}
		rows = append(rows, table.Row{
			s.Name,
			s.Status,
			fmtDuration(time.Duration(s.Duration * 1000 * 1000)),
			s.ID,
		})
	}

	m.table.SetRows(rows)
	var newCols []table.Column

	newCols = append(newCols, columns...)

	if m.sort != none {
		if m.asc {
			newCols[int64(m.sort)].Title = newCols[int64(m.sort)].Title + " ↓"
		} else {
			newCols[int64(m.sort)].Title = newCols[int64(m.sort)].Title + " ↑"
		}
	}
	m.table.SetColumns(newCols)
}

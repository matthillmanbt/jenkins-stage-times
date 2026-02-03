# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A CLI tool for analyzing Jenkins pipeline execution times and stages. Built with Go using Cobra for CLI commands and Bubble Tea for interactive TUI components.

## Environment Setup

Required environment variables:
```bash
export JENKINS_HOST=https://<host>
export JENKINS_USER=<jenkins user>
export JENKINS_KEY=<api key for that jenkins user>
```

Configuration can also be stored in `~/.jenkins.yaml` instead of environment variables.

## Common Commands

### Build
```bash
go build -o jenkins .
```

### Run locally
```bash
go run main.go <command>
```

### Install dependencies
```bash
go mod tidy
```

### Release
Uses GoReleaser for building releases (automated via GitHub Actions). The binary is named `jenkins` and version info is injected via ldflags: `-s -w -X jenkins/cmd.Version={{.Version}}`

## Architecture

### Module Structure
- **Module name**: `jenkins`
- **Entry point**: `main.go` - minimal, just calls `cmd.Execute()`
- **Commands**: All command logic lives in `cmd/` package

### Command Architecture
Built on Cobra with the following command structure:
- **Default command**: `timing` (automatically invoked if no subcommand specified)
- **Root command** (`cmd/root.go`): Defines persistent flags (host, user, key, pipeline, verbose) and configuration loading via Viper
- **Commands register themselves** in their `init()` functions by calling `rootCmd.AddCommand()`

### Key Commands
- `timing`: Calculates average/min/max stage times across recent successful builds (default command)
- `stages [build_id]`: Interactive TUI to browse builds and drill into stage logs
- `monitor`: Real-time monitoring of Jenkins jobs
- `push`: Triggers Jenkins builds
- `status`: Shows current build status
- `log`: Retrieves build logs
- `open`: Opens Jenkins URLs in browser
- `latest`: Gets latest build information

### Data Flow
1. Commands use `jenkinsRequest()` in `cmd/request.go` to make authenticated API calls
2. Responses are decoded into Go structs defined in `cmd/jenkins.go`:
   - `Job`: Top-level build with stages
   - `Stage`: Pipeline stage with timing and nested StageFlowNodes
   - `WorkflowRun`: Build metadata
   - `Node`: Console log output
3. Uses Jenkins workflow API endpoints (e.g., `/wfapi/runs`, `/wfapi/describe`)

### Interactive TUI (Bubble Tea)
The `stages` command implements a multi-level interactive browser:
- **Model state**: Tracks current view context (job list → job stages → stage details → console logs)
- **Navigation**: Enter/right drills down, Esc/left goes back, q quits
- **Features**: Sortable columns (n/s/d/t keys), filterable stage lists (f key), scrollable console output
- **Components**: Uses bubbles table, viewport, and textinput

### Styling
Consistent styling defined in `cmd/util.go`:
- Orange (`#F86600`) as primary accent color
- Cyan for selections
- Gray for backgrounds and muted text
- Lipgloss for all text styling and layout

### Configuration
- Viper handles config loading with precedence: flags > env vars > config file
- Environment variables prefixed with `JENKINS_` (e.g., `JENKINS_HOST`)
- Default pipeline is "master" if not specified

## Notable Patterns

- **Default command injection**: `setDefaultCommandIfNonePresent()` in root.go manipulates os.Args to set default command
- **Verbose logging**: Uses `-v` flag count for verbosity levels (verbose/vVerbose functions)
- **Stage filtering**: `timing` command supports multiple filters with AND/OR logic
- **Type-safe messages**: Bubble Tea uses Go structs (Job, Stage, Node) as message types for state updates

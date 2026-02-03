# Jenkins Build Management Skill

Comprehensive Jenkins integration for triggering builds, monitoring progress, and diagnosing failures.

## Description

This skill provides Claude Code with the ability to interact with Jenkins CI/CD pipelines:
- **Start builds** with parameterized configurations
- **Monitor builds** in the background for long-running jobs (20-30+ minutes)
- **Diagnose failures** by identifying failed stages and analyzing their logs
- **Query build status** and get detailed stage information
- **View logs** for specific stages to troubleshoot issues

Perfect for repos that need to trigger Jenkins builds, wait for results, and analyze failures automatically.

## Prerequisites

1. **Jenkins credentials** configured via environment variables:
   ```bash
   export JENKINS_HOST=https://your-jenkins-server.com
   export JENKINS_USER=your-username
   export JENKINS_KEY=your-api-key
   ```

2. **Jenkins binary** installed:
   - Build from source: `go install` in this repo
   - Or copy the compiled `jenkins` binary to your PATH

## Configuration

Create `~/.jenkins.yaml` with your defaults:

```yaml
host: https://jenkins.example.com
user: your-username
key: your-api-key
pipeline: master  # default pipeline name

# Optional: Product configurations (for multi-product repos)
products:
  rs:
    search_name: "ingredi"
    display_name: "RS"
  pra:
    search_name: "bpam"
    display_name: "PRA"

deployment:
  domain: "dev.example.com"
```

## Commands

### Start a Build

```bash
# Trigger a build and monitor until completion
jenkins push <build_number_or_product> <subdomain>

# Monitor an existing build
jenkins monitor <build_id> [<build_id2>...]
```

### Diagnose Build Failures

```bash
# Comprehensive analysis: show all failed stages with logs
jenkins diagnose <build_id>

# Show more log context
jenkins diagnose <build_id> --log-lines 100

# Show all stages (not just failures)
jenkins diagnose <build_id> --all
```

### Query Build Information

```bash
# List all failed stages
jenkins failed <build_id>

# Get log for a specific stage
jenkins stage-log <build_id> <stage_id>

# Show only last 50 lines of a stage log
jenkins stage-log <build_id> <stage_id> --tail 50

# Get build status
jenkins status -b <build_id>
```

### Analyze Performance

```bash
# Show average stage times across recent builds
jenkins timing

# Filter specific stages
jenkins timing -f "Build" -f "Test"

# Show longest stage executions
jenkins timing --longest
```

## Typical Workflows

### Workflow 1: Trigger and Monitor a Build

```bash
# Start the build
jenkins push 1234 dev-testing

# It automatically monitors in background
# You'll get a notification when complete
```

### Workflow 2: Investigate a Failed Build

```bash
# Get comprehensive diagnosis
jenkins diagnose 5678

# This shows:
# - Build summary (status, duration, URL)
# - List of failed stages
# - Logs for each failed stage
# - Summary for AI analysis
```

### Workflow 3: Deep Dive on Specific Stage

```bash
# List all failed stages
jenkins failed 5678

# Get full log for specific stage
jenkins stage-log 5678 stage-id-123
```

## Integration with Claude Code

When used as a skill in other repos:

1. **Claude can trigger builds** when you ask to "start a build" or "deploy to dev"
2. **Claude monitors builds** in the background and notifies you when done
3. **Claude diagnoses failures** automatically, showing you exactly what went wrong
4. **Claude analyzes logs** to suggest fixes for common failure patterns

## Examples

### Example 1: Ask Claude to start a build

```
You: "Start a build for RS and deploy it to dev-testing"

Claude uses: jenkins push rs dev-testing
Claude monitors the build and notifies you when complete
```

### Example 2: Ask Claude to investigate a failure

```
You: "Build 1234 failed, can you figure out why?"

Claude uses: jenkins diagnose 1234
Claude analyzes the output and tells you:
- Which stages failed
- What errors were in the logs
- Potential causes and fixes
```

### Example 3: Monitor long-running build

```
You: "Monitor build 5678 and let me know when it's done"

Claude uses: jenkins monitor 5678
Claude continues in background, notifies when complete (20-30 min later)
```

## Error Handling

The skill provides helpful error messages:
- Missing credentials → Shows how to set JENKINS_* env vars
- Build not found → Suggests checking pipeline name
- API errors → Shows status code and URL for debugging

## Advanced Usage

### Multiple Pipelines

```bash
# Use different pipeline
jenkins --pipeline production status -b 1234

# Override in config file
jenkins --config ~/.jenkins-prod.yaml diagnose 5678
```

### Verbose Output

```bash
# Debug mode
jenkins -vv diagnose 1234

# Shows:
# - API requests being made
# - Response parsing details
# - Timing information
```

## Output Formats

All commands provide:
- **Human-readable** output with colors and formatting
- **Structured data** that's easy for AI to parse
- **Actionable information** (stage IDs, build URLs, log excerpts)

## Performance

- **Fast queries**: Most commands complete in <2 seconds
- **Background monitoring**: Doesn't block Claude or terminal
- **Efficient log fetching**: Only fetches what's needed
- **Smart defaults**: Shows most relevant information first

## Troubleshooting

**"authentication error"**
→ Check JENKINS_HOST, JENKINS_USER, JENKINS_KEY env vars

**"build not found"**
→ Verify pipeline name with `--pipeline` flag

**"no log available"**
→ Some stages don't produce logs (e.g., parallel wrappers)

**"stage not found"**
→ Use `jenkins failed <build_id>` to list all failed stages and their IDs

## Notes

- Build monitoring happens in background (perfect for long builds)
- All stage logs are available for AI analysis
- Recursive stage search (finds failures in nested parallel stages)
- Smart log truncation (shows first/last lines of very long logs)
- Color-coded output (green=success, red=failure, orange=warning)

# Jenkins Skill for Claude Code

A comprehensive Jenkins integration skill that enables Claude Code to trigger builds, monitor progress, and diagnose failures across any repository that uses Jenkins.

## Quick Start

### 1. Install the Skill

From this repository:
```bash
cd .claude/skills/jenkins
./install.sh
```

Or manually:
```bash
go build -o jenkins .
sudo cp jenkins /usr/local/bin/
```

### 2. Configure Credentials

```bash
export JENKINS_HOST=https://jenkins.example.com
export JENKINS_USER=your-username
export JENKINS_KEY=your-api-key
```

Or create `~/.jenkins.yaml`:
```yaml
host: https://jenkins.example.com
user: your-username
key: your-api-key
pipeline: master
```

### 3. Use with Claude Code

In any repository with this skill available, you can ask Claude:

- **"Start a build for development"**
- **"Monitor build 1234"**
- **"Why did build 5678 fail?"**
- **"Show me the logs for the failing test stage"**
- **"What's the average build time?"**

Claude will use the appropriate jenkins commands automatically.

## Key Features

### 🚀 Start & Monitor Builds
- Trigger builds with parameters
- Monitor in background for 20-30+ minute builds
- Get notified when complete

### 🔍 Diagnose Failures
- Automatically identify failed stages
- Show relevant log excerpts
- Recursive search through nested stages

### 📊 Analyze Performance
- Average/min/max stage durations
- Identify slow stages
- Track performance trends

### 🛠️ Debug Tools
- Get logs for specific stages
- Interactive stage browser (TUI)
- Tail/head options for long logs

## Commands Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `push` | Trigger build | `jenkins push 1234 dev-test` |
| `monitor` | Watch build progress | `jenkins monitor 5678` |
| `diagnose` | Analyze failures | `jenkins diagnose 5678` |
| `failed` | List failed stages | `jenkins failed 5678` |
| `stage-log` | Get stage logs | `jenkins stage-log 5678 stage-id` |
| `stages` | Interactive browser | `jenkins stages 5678` |
| `status` | Build status | `jenkins status -b 5678` |
| `timing` | Performance analysis | `jenkins timing --longest` |

## Use Cases

### Development Workflow
1. Make code changes
2. Ask Claude: "start a build"
3. Claude triggers and monitors
4. Get notification when complete
5. If failed, Claude diagnoses automatically

### Troubleshooting
1. Notice a failed build
2. Ask Claude: "why did build X fail?"
3. Claude runs `diagnose` and analyzes logs
4. Claude suggests potential fixes
5. Apply fixes and restart

### Performance Analysis
1. Ask Claude: "which stages are slowest?"
2. Claude runs `timing --longest`
3. Review bottlenecks
4. Optimize slow stages

## Example Interactions

**User:** Start a build for RS and deploy to dev-testing

**Claude:**
```
I'll trigger a build for RS and deploy it to dev-testing.

Running: jenkins push rs dev-testing

Build started successfully! Monitoring in background...
Build #5678 is now running. I'll notify you when it completes.
```

---

**User:** Build 1234 failed, can you see why?

**Claude:**
```
Let me analyze build 1234 to see what went wrong.

Running: jenkins diagnose 1234

BUILD DIAGNOSIS: Build #1234 FAILED

Failed stages:
1. Unit Tests (Duration: 02:34.567)
2. Integration Tests (Duration: 05:12.890)

Analyzing the logs, I see:
- Unit Tests failed with "NullPointerException in UserService.java:45"
- Integration Tests failed due to database connection timeout

The root cause appears to be the NullPointerException. Here's the relevant code:
[shows log excerpt]

I recommend checking UserService.java line 45 for a null check before accessing the user object.
```

## Advanced Configuration

### Multiple Jenkins Servers

Create separate config files:
```bash
# ~/.jenkins-dev.yaml
host: https://jenkins-dev.example.com
pipeline: development

# ~/.jenkins-prod.yaml
host: https://jenkins-prod.example.com
pipeline: production
```

Use with: `jenkins --config ~/.jenkins-prod.yaml diagnose 1234`

### Custom Defaults

```yaml
# ~/.jenkins.yaml
pipeline: master
products:
  myapp:
    search_name: "myapp-backend"
    display_name: "MyApp"
deployment:
  domain: "staging.myapp.com"
```

## Integration

### In Your Repo

Copy this skill to your repo:
```bash
cp -r .claude/skills/jenkins /path/to/your/repo/.claude/skills/
```

Claude Code will automatically detect and use the skill when you ask Jenkins-related questions.

### Programmatic Use

The CLI can be used in scripts:
```bash
#!/bin/bash
# Trigger build and wait
build_id=$(jenkins push main staging | grep "Build" | awk '{print $2}')
jenkins monitor "$build_id"

# Check if it succeeded
if jenkins status -b "$build_id" | grep -q "SUCCESS"; then
    echo "Build passed!"
else
    # Diagnose and email logs
    jenkins diagnose "$build_id" | mail -s "Build Failed" team@example.com
fi
```

## Troubleshooting

**Skill not detected by Claude Code:**
- Ensure `.claude/skills/jenkins/skill.md` exists
- Restart Claude Code

**Authentication errors:**
- Verify JENKINS_* environment variables
- Test with: `jenkins --host https://... --user ... --key ... status -b 1`

**Build not found:**
- Check pipeline name: `jenkins --pipeline production status -b 1234`
- List recent builds: `jenkins stages`

**Performance issues:**
- Use `--tail` flag to limit log output: `jenkins stage-log 1234 stage-id --tail 50`
- Increase verbosity to debug: `jenkins -vv diagnose 1234`

## Contributing

To improve this skill:
1. Add new commands in `cmd/`
2. Update `skill.md` documentation
3. Test with various Jenkins configurations
4. Submit PR with examples

## License

Same license as the jenkins-stage-times repository.

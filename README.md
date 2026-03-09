<div align="center">

# ttg

**A fast, feature-rich CLI for [TickTick](https://ticktick.com) — built in Go.**

Add tasks, manage checklists, set reminders, search, and check off your day without leaving the terminal.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![ClawHub](https://img.shields.io/badge/ClawHub-ticktick--go-orange?style=flat-square)](https://clawhub.com/skills/ticktick-go)

</div>

---

## Features

- ✅ **Full task CRUD** — list, add, edit, complete, delete
- 📁 **Project management** — list projects with task counts
- ☑️ **Checklists & subtasks** — create checklist tasks, add/complete/delete items
- ⏰ **Reminders** — `15m`, `1h`, `1d`, `on-time` (comma-separated for multiple)
- 🔁 **Recurring tasks** — `daily`, `weekly`, `monthly`, `yearly`, or custom RRULE
- 🔍 **Search** — find tasks by title across all projects
- 🏷️ **Tags** — add tags to tasks, filter by tag, list all tags
- 🗓️ **Natural language dates** — `tomorrow 3pm`, `next monday`, `in 2 days`
- 🔺 **Priority** — `low`, `medium`, `high` with shorthand flags (`--high`, `--med`, `--low`)
- 📊 **Progress display** — visual progress bars for checklist tasks (0–100%)
- ⚡ **Quick-add shorthands** — `--today`, `--tomorrow`/`--tmrw` for fast capture
- 📤 **JSON output** — pipe any list into `jq` for scripting
- 🔐 **OAuth2 auth** — secure login via TickTick Open API

---

## Installation

**From source (macOS, Linux, any platform):**

```bash
git clone https://github.com/dhruvkelawala/ticktick-go
cd ticktick-go
make install   # builds and copies to ~/.local/bin/ttg
```

Make sure `~/.local/bin` is on your `$PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

> **Platform note:** `make install` compiles from source and works on macOS (Intel + Apple Silicon), Linux (amd64, arm64), and any Go-supported platform. Requires Go 1.21+.

---

## Setup

1. Register an app at [developer.ticktick.com](https://developer.ticktick.com/manage) to get your **Client ID** and **Client Secret**.

2. Create `~/.config/ttg/config.json`:

```json
{
  "client_id": "YOUR_CLIENT_ID",
  "client_secret": "YOUR_CLIENT_SECRET",
  "timezone": "Europe/London"
}
```

3. Authenticate:

```bash
ttg auth login      # opens browser for OAuth2
ttg auth status     # verify you're logged in
ttg auth logout     # clear stored token
```

Token is stored at `~/.config/ttg/token.json`.

---

## Usage

### Tasks

```bash
# List
ttg task list                              # Inbox (default)
ttg task list --all                        # All tasks across all projects
ttg task list --project "Work"             # By project name
ttg task list --due today                  # Due today
ttg task list --due overdue                # Overdue tasks
ttg task list --priority high              # By priority
ttg task list --tag "urgent"               # By tag
ttg task list --completed                  # Show completed tasks
ttg task list --json                       # JSON output for scripting

# Add
ttg task add "Buy milk"
ttg task add "Ship feature" --project "Work" --priority high --due "tomorrow 9am"
ttg task add "Call dentist" --today --high --remind "1h,on-time"
ttg task add "Weekly review" --due "next friday" --repeat weekly
ttg task add "Quick note" -n "Don't forget the attachment" --tag "work,followup"

# Quick-add shorthands
ttg task add "Morning standup" --today --med
ttg task add "Submit report" --tomorrow --high
ttg task add "Urgent fix" --tmrw --remind "15m"

# View / manage
ttg task get <id>                          # Full task details
ttg task done <id>                         # Mark complete
ttg task delete <id>                       # Delete

# Edit
ttg task edit <id> --title "Updated title"
ttg task edit <id> --priority medium --due "next monday"
ttg task edit <id> --remind "1h,15m" --repeat monthly
ttg task edit <id> --tag "work,important" --start "tomorrow 9am"

# Search
ttg task search "deploy"                   # Search tasks by title
```

### Checklists & Subtasks

```bash
# Create a checklist task
ttg task add "Pack for trip" --checklist --items "Passport,Charger,Clothes"

# Manage checklist items
ttg task items <task-id>                   # List all items
ttg task item-add <task-id> "Toothbrush"   # Add an item
ttg task item-done <task-id> <item-id>     # Complete an item
ttg task item-delete <task-id> <item-id>   # Delete an item

# Convert existing task to checklist
ttg task edit <id> --kind checklist
```

Checklist tasks show a visual progress bar in list and detail views:
```
☑️ Pack for trip [60%]
│ Progress: [██████░░░░] 60%
```

### Projects

```bash
ttg project list                           # All projects with task counts
ttg project get <id>                       # Project details
```

### Tags

```bash
ttg tag list                               # List all tags used across tasks
```

---

## Reminders

Add one or more reminders with `--remind` (comma-separated):

| Shorthand | Meaning |
|-----------|---------|
| `on-time` | At the due time |
| `5m` | 5 minutes before |
| `15m` | 15 minutes before |
| `30m` | 30 minutes before |
| `1h` | 1 hour before |
| `1d` | 1 day before |

```bash
ttg task add "Meeting" --due "3pm" --remind "15m,on-time"
ttg task edit <id> --remind "1h,30m"
```

## Recurring Tasks

| Pattern | Meaning |
|---------|---------|
| `daily` | Every day |
| `weekly` | Every week |
| `monthly` | Every month |
| `yearly` | Every year |
| `RRULE:...` | Custom iCal RRULE |

```bash
ttg task add "Daily standup" --due "9am" --repeat daily
ttg task add "Monthly review" --due "1st" --repeat monthly
```

## Due Date Formats

| Input | Meaning |
|-------|---------|
| `today`, `tomorrow` | Midnight of that day |
| `next monday` | Following Monday |
| `3pm`, `tomorrow 3pm` | Specific time |
| `in 2 days`, `in 3 hours` | Relative offset |
| `2026-03-20` | ISO date |
| `2026-03-20T15:00:00` | ISO datetime |

## Priority

`none` (default) · `low` · `medium` · `high`

Shorthand flags: `--high`, `--med`/`--medium`, `--low`

---

## JSON / Scripting

Any list command accepts `--json` / `-j`:

```bash
# Get all task titles
ttg task list --all --json | jq '.[].title'

# Find overdue high-priority tasks
ttg task list --due overdue --priority high --json

# Export project task counts
ttg project list --json | jq '.[] | {name, taskCount}'
```

---

## AI Agent Integration

ttg is available as an [OpenClaw](https://openclaw.ai) agent skill on [ClawHub](https://clawhub.com/skills/ticktick-go):

```bash
clawhub install ticktick-go
```

This lets AI agents manage your TickTick tasks via natural language.

---

## License

MIT

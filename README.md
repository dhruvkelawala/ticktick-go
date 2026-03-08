<div align="center">

# tt

**A fast, minimal CLI for [TickTick](https://ticktick.com) — built in Go.**

Add tasks, manage projects, and check off your day without leaving the terminal.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

</div>

---

## Features

- ✅ Full task CRUD — list, add, edit, complete, delete
- 📁 Project management
- 🗓 Natural language due dates (`tomorrow 3pm`, `next monday`, `in 2 days`)
- 🔺 Priority support (`low`, `medium`, `high`)
- 📤 JSON output for scripting
- 🔐 OAuth2 auth via TickTick Open API

---

## Installation

**From source:**

```bash
git clone https://github.com/dhruvkelawala/tt
cd tt
make install   # builds and copies to ~/.local/bin/ttg
```

Make sure `~/.local/bin` is on your `$PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

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
tt auth login
```

This opens a browser for OAuth2 and stores your token at `~/.config/ttg/token.json`.

---

## Usage

### Tasks

```bash
tt task list                          # Inbox (default)
tt task list --all                    # All tasks
tt task list --project "Work"         # By project
tt task list --due today              # Due today
tt task list --priority high          # By priority
tt task list --json                   # JSON output

tt task add "Buy milk"
tt task add "Ship feature" --project "Work" --priority high --due "tomorrow 9am"

tt task get <id>                      # Task details
tt task done <id>                     # Mark complete
tt task delete <id>                   # Delete
tt task edit <id> --title "New title" --priority medium
```

### Projects

```bash
tt project list
tt project get <id>
```

### JSON / scripting

Any command accepts `--json` / `-j`:

```bash
tt task list --json | jq '.[].title'
```

---

## Due Date Formats

| Input | Meaning |
|-------|---------|
| `today`, `tomorrow` | Midnight of that day |
| `next monday` | Following Monday |
| `3pm`, `tomorrow 3pm` | Specific time |
| `in 2 days`, `in 3 hours` | Relative offset |
| `2026-03-20` | ISO date |
| `2026-03-20T15:00:00` | ISO datetime |

---

## Priority

`none` (default) · `low` · `medium` · `high`

---

## License

MIT

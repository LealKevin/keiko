# Keiko 稽古

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
[![GitHub stars](https://img.shields.io/github/stars/LealKevin/keiko?style=flat)](https://github.com/LealKevin/keiko/stargazers)

Learn Japanese while you code. No context switching.

![Status Bar Demo](docs/demo.gif)

Keiko puts Japanese flashcards in your terminal status bar and lets you review with global hotkeys - learn vocabulary without leaving your editor.

## Features

**Status Bar Mode** - Flashcards in your tmux status bar
- Review words while coding with F-key hotkeys
- Syncs with your Anki decks
- Shows due count in real-time

**News Reader** - Read NHK Easy News with AI-powered assistance
- Token-by-token navigation (vim-style: hjkl)
- See readings, base forms, and translations for every word
- Track what you've read

**Built-in Vocabulary** - 5,000+ JLPT words (N5-N1)
- Spaced repetition built in
- Filter by JLPT level

## Quick Start

### Prerequisites

- Go 1.21+
- tmux (for status bar mode)
- [Anki](https://apps.ankiweb.net/) + [AnkiConnect](https://ankiweb.net/shared/info/2055492159) (optional, for Anki sync)

### Install

```bash
go install github.com/LealKevin/keiko@latest
```

Or build from source:

```bash
git clone https://github.com/LealKevin/keiko.git
cd keiko
go build -o keiko ./cmd/main.go
```

### Run

**Status bar mode** (default):
```bash
# Add to your tmux status bar
set -g status-right '#(keiko)'

# Or run standalone
keiko
```

**TUI mode**:
```bash
keiko --tui
```

## Hotkeys

| Key | Action |
|-----|--------|
| F2 | Open settings (TUI popup) |
| F3 | Toggle Vocab/Anki mode |
| F4 | Reveal answer |
| F5 | Again (mark for review) |
| F6 | Good (advance card) |

## TUI Navigation

| Key | Action |
|-----|--------|
| Tab | Switch tabs (News / Settings) |
| j/k | Navigate list |
| h/l | Navigate tokens in article |
| Enter | Open article |
| Esc | Back |
| q | Quit |

## Configuration

Config file: `~/.config/keiko/config.yaml`

```yaml
loop_interval: 10          # Seconds between card changes
show_furigana: true        # Show readings above kanji
show_translation: true     # Show English meanings
show_jlpt_level: true      # Show N1-N5 level
jlpt_levels: [5, 4, 3]     # Which levels to study
anki_deck: "Core2k"        # Your Anki deck name
news_server_url: "..."     # News API endpoint
```

## Screenshots

### Status Bar
```
[Core2k: 12 due] 食べる → [F4]              # Question
[Core2k: 11 due] 食べる - to eat → [F5|F6]  # Answer revealed
```

### TUI News Reader
```
┌─ Articles ─────────┬─ 日本で大きな地震 ─────────────────┐
│ ● 日本で大きな地震 │                                    │
│   新しい法律が...   │ 日本で 大きな 地震が ありました。  │
│   東京の天気...     │       ↑                            │
│                     │ [おおきな] 大きい - big, large     │
└─────────────────────┴────────────────────────────────────┘
```

## How It Works

1. **Vocabulary Mode**: Built-in JLPT vocabulary with spaced repetition
2. **Anki Mode**: Syncs with AnkiConnect to use your existing decks
3. **News Mode**: Fetches NHK Easy News, tokenizes with Gemini AI for morphological analysis

## Tech Stack

- [Go](https://go.dev)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [gohook](https://github.com/robotn/gohook) - Global hotkeys
- [Gemini API](https://ai.google.dev/) - Japanese tokenization
- SQLite - Local vocabulary database

## License

MIT

## Contributing

Issues and PRs welcome.

---

If you find Keiko useful, consider [giving it a star](https://github.com/LealKevin/keiko)!

# Design: `cast voices pick` — Interactive Voice Selector

## Overview

Terminal-based interactive UI for browsing and selecting Typecast voices with real-time filtering, preview playback, and quick actions. Built with bubbletea.

## Entry

```bash
cast voices pick                          # Full list
cast voices pick --gender female          # Pre-filtered
cast voices pick --age young_adult        # Reuses existing list filters
cast voices pick --text "Hello world"     # Custom preview text
```

Reuses all existing `voices list` filter flags: `--model`, `--gender`, `--age`, `--use-case`.

## UI Layout

```
> filter: 수진_
  2/127

> 수진    female  young_adult  Conversational,News
  수진B   female  teenager     Anime,Game

  [P] Preview  [S] Set default  [C] Copy ID  [Enter] Select  [Esc] Quit
```

- Top: real-time text filter input
- Middle: voice list showing name, gender, age, use_cases
- Bottom: shortcut key legend

## Keyboard Actions

| Key     | Action                                          |
|---------|-------------------------------------------------|
| typing  | Real-time filter by voice name                  |
| Up/Down | Navigate voice list                             |
| P       | Preview selected voice via TTS + audio playback |
| S       | Save selected voice as default in config        |
| C       | Copy voice ID to clipboard                      |
| Enter   | Print voice ID to stdout and exit               |
| Esc / q | Cancel and exit                                 |

## Preview (P key)

- Uses `--text` flag value if provided
- Falls back to default: `"The quick brown fox jumps over the lazy dog."`
- Reuses existing TTS API client and platform-specific audio playback

## Filtering

1. API-level pre-filtering via flags (gender, age, model, use_case) — reduces initial list
2. Client-side real-time filtering by voice name substring as user types

## Technical Stack

- **bubbletea** (charmbracelet/bubbletea) — TUI framework
- **bubbles** (charmbracelet/bubbles) — text input, list components
- **lipgloss** (charmbracelet/lipgloss) — styling

## Architecture

- New file: `cmd/pick.go` — bubbletea model, view, update logic
- Subcommand of `voices`: `cast voices pick`
- Reuses `internal/client` for API calls and `cmd/audio.go` for playback
- Clipboard: `pbcopy` (macOS), `xclip`/`xsel` (Linux), `clip` (Windows)

## Scope

This feature adds one new subcommand. No changes to existing commands or API client logic.

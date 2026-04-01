# `cast voices pick` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an interactive terminal UI for browsing and selecting Typecast voices with real-time filtering, preview, and quick actions.

**Architecture:** New `cmd/pick.go` file registers a `pick` subcommand under `voices`. It uses bubbletea for the interactive TUI, reuses the existing API client for voice listing and TTS preview, and reuses existing audio playback and config persistence. Platform-specific clipboard handled via build tags.

**Tech Stack:** bubbletea, bubbles (textinput, list), lipgloss

---

### Task 1: Add bubbletea dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add bubbletea, bubbles, and lipgloss**

```bash
cd /Users/ironyee/Documents/cast
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Verify build still works**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add bubbletea, bubbles, lipgloss for interactive voice picker"
```

---

### Task 2: Implement clipboard helpers (platform-specific)

**Files:**
- Create: `cmd/clipboard.go`
- Create: `cmd/clipboard_darwin.go`
- Create: `cmd/clipboard_linux.go`
- Create: `cmd/clipboard_windows.go`
- Create: `cmd/clipboard_test.go`

- [ ] **Step 1: Write tests for clipboard**

Create `cmd/clipboard_test.go`:

```go
package cmd

import (
	"testing"
)

func TestCopyToClipboard(t *testing.T) {
	// Smoke test: should not panic or error on the current platform.
	// We can't reliably assert clipboard contents in CI, but we can
	// verify the function runs without error.
	err := copyToClipboard("test-voice-id")
	if err != nil {
		t.Skipf("clipboard not available: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/ -run TestCopyToClipboard -v
```

Expected: FAIL — `copyToClipboard` undefined

- [ ] **Step 3: Create platform-specific clipboard implementations**

Create `cmd/clipboard_darwin.go`:

```go
//go:build darwin

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
```

Create `cmd/clipboard_linux.go`:

```go
//go:build linux

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	// Try xclip first, fall back to xsel.
	for _, bin := range []string{"xclip", "xsel"} {
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		var cmd *exec.Cmd
		if bin == "xclip" {
			cmd = exec.Command(path, "-selection", "clipboard")
		} else {
			cmd = exec.Command(path, "--clipboard", "--input")
		}
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return exec.ErrNotFound
}
```

Create `cmd/clipboard_windows.go`:

```go
//go:build windows

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("clip")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./cmd/ -run TestCopyToClipboard -v
```

Expected: PASS (or SKIP if no clipboard available)

- [ ] **Step 5: Commit**

```bash
git add cmd/clipboard_darwin.go cmd/clipboard_linux.go cmd/clipboard_windows.go cmd/clipboard_test.go
git commit -m "feat: add platform-specific clipboard helpers"
```

---

### Task 3: Implement the pick command with bubbletea TUI

**Files:**
- Create: `cmd/pick.go`
- Modify: `cmd/voices.go` (register subcommand)

- [ ] **Step 1: Create `cmd/pick.go` with the full bubbletea model**

Create `cmd/pick.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/neosapience/cast/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultPreviewText = "The quick brown fox jumps over the lazy dog."

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Interactively browse and select a voice",
	RunE: func(cmd *cobra.Command, args []string) error {
		previewText, _ := cmd.Flags().GetString("text")
		if previewText == "" {
			previewText = defaultPreviewText
		}

		flags := cmd.Flags()
		model, _ := flags.GetString("model")
		gender, _ := flags.GetString("gender")
		age, _ := flags.GetString("age")
		useCase, _ := flags.GetString("use-case")

		baseURL := viper.GetString("base_url")
		var c *client.Client
		if baseURL != "" {
			c = client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
		} else {
			c = client.New(viper.GetString("api_key"))
		}

		voices, err := c.ListVoices(client.ListVoicesParams{
			Model:   model,
			Gender:  gender,
			Age:     age,
			UseCase: useCase,
		})
		if err != nil {
			return err
		}

		if len(voices) == 0 {
			return fmt.Errorf("no voices found matching the given filters")
		}

		m := newPickModel(voices, c, previewText)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm := finalModel.(pickModel)
		if fm.selected != nil {
			fmt.Println(fm.selected.VoiceID)
		}
		if fm.err != nil {
			return fm.err
		}
		return nil
	},
}

// Styles
var (
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type pickModel struct {
	textInput   textinput.Model
	voices      []client.Voice
	filtered    []client.Voice
	cursor      int
	selected    *client.Voice
	client      *client.Client
	previewText string
	status      string
	err         error
	width       int
	height      int
}

func newPickModel(voices []client.Voice, c *client.Client, previewText string) pickModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter by name..."
	ti.Focus()

	return pickModel{
		textInput:   ti,
		voices:      voices,
		filtered:    voices,
		client:      c,
		previewText: previewText,
	}
}

func (m pickModel) Init() tea.Cmd {
	return textinput.Blink
}

type statusMsg string

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				v := m.filtered[m.cursor]
				m.selected = &v
			}
			return m, tea.Quit

		case tea.KeyRunes:
			r := string(msg.Runes)
			// Handle single-key shortcuts only when the character would be
			// the only content in the input (i.e. the filter is currently empty).
			if m.textInput.Value() == "" {
				switch strings.ToLower(r) {
				case "p":
					return m, m.previewVoice()
				case "s":
					return m, m.setDefault()
				case "c":
					return m, m.copyID()
				}
			}
		}
	}

	// Update text input and refilter.
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m *pickModel) applyFilter() {
	query := strings.ToLower(m.textInput.Value())
	if query == "" {
		m.filtered = m.voices
	} else {
		m.filtered = m.filtered[:0]
		for _, v := range m.voices {
			if strings.Contains(strings.ToLower(v.VoiceName), query) {
				m.filtered = append(m.filtered, v)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m pickModel) View() string {
	var b strings.Builder

	// Filter input.
	b.WriteString(filterStyle.Render("> "))
	b.WriteString(m.textInput.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d", len(m.filtered), len(m.voices))))
	b.WriteString("\n\n")

	// Voice list — show up to the available terminal height minus chrome.
	maxVisible := m.height - 7 // filter(1) + count(1) + blank(1) + status(1) + help(2) + padding(1)
	if maxVisible < 3 {
		maxVisible = 3
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		v := m.filtered[i]
		useCases := strings.Join(v.UseCases, ",")
		line := fmt.Sprintf("%-12s %-8s %-14s %s", v.VoiceName, v.Gender, v.Age, useCases)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	// Status message.
	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render("  " + m.status))
	}

	// Help bar.
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  [P] Preview  [S] Set default  [C] Copy ID  [Enter] Select  [Esc] Quit"))
	b.WriteString("\n")

	return b.String()
}

func (m pickModel) previewVoice() tea.Cmd {
	return func() tea.Msg {
		if len(m.filtered) == 0 {
			return statusMsg("no voice to preview")
		}
		v := m.filtered[m.cursor]
		req := client.TTSRequest{
			VoiceID: v.VoiceID,
			Text:    m.previewText,
			Model:   defaultModel,
		}
		audio, err := m.client.TextToSpeech(req)
		if err != nil {
			return statusMsg(fmt.Sprintf("preview failed: %v", err))
		}
		if err := playAudio(audio, "wav"); err != nil {
			return statusMsg(fmt.Sprintf("playback failed: %v", err))
		}
		return statusMsg(fmt.Sprintf("previewed: %s", v.VoiceName))
	}
}

func (m pickModel) setDefault() tea.Cmd {
	return func() tea.Msg {
		if len(m.filtered) == 0 {
			return statusMsg("no voice to set")
		}
		v := m.filtered[m.cursor]
		config, err := readConfig()
		if err != nil {
			return statusMsg(fmt.Sprintf("config error: %v", err))
		}
		config["voice_id"] = v.VoiceID
		if err := writeConfig(config); err != nil {
			return statusMsg(fmt.Sprintf("save failed: %v", err))
		}
		return statusMsg(fmt.Sprintf("default voice set to %s (%s)", v.VoiceName, v.VoiceID))
	}
}

func (m pickModel) copyID() tea.Cmd {
	return func() tea.Msg {
		if len(m.filtered) == 0 {
			return statusMsg("no voice to copy")
		}
		v := m.filtered[m.cursor]
		if err := copyToClipboard(v.VoiceID); err != nil {
			return statusMsg(fmt.Sprintf("copy failed: %v", err))
		}
		return statusMsg(fmt.Sprintf("copied: %s", v.VoiceID))
	}
}

func init() {
	pickCmd.Flags().String("text", "", "Custom text for voice preview (default: English sample sentence)")
	pickCmd.Flags().String("model", "", "Filter by model (ssfm-v30, ssfm-v21)")
	pickCmd.Flags().String("gender", "", "Filter by gender (male, female)")
	pickCmd.Flags().String("age", "", "Filter by age (child, teenager, young_adult, middle_age, elder)")
	pickCmd.Flags().String("use-case", "", "Filter by use case")
}
```

- [ ] **Step 2: Register pick subcommand in voices.go**

In `cmd/voices.go`, add `voicesCmd.AddCommand(pickCmd)` inside `init()`:

```go
// In cmd/voices.go init(), add this line after the existing AddCommand calls:
voicesCmd.AddCommand(pickCmd)
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/pick.go cmd/voices.go
git commit -m "feat: add interactive voice picker (cast voices pick)"
```

---

### Task 4: Fix shortcut key handling to work with non-empty filter

The current implementation only triggers P/S/C shortcuts when the filter input is empty. This is intentional — when the user is typing a filter like "park", pressing P should type "p" into the filter, not trigger preview. The shortcuts activate when the filter is cleared (empty). This is a reasonable UX trade-off for a text-based filter UI.

However, we should document this behavior and add Ctrl-key alternatives that work regardless of filter state.

**Files:**
- Modify: `cmd/pick.go`

- [ ] **Step 1: Add Ctrl-key shortcuts as alternatives**

In the `Update` method of `cmd/pick.go`, add Ctrl-key handling before the `tea.KeyRunes` case:

```go
		// Ctrl-key shortcuts work regardless of filter state.
		case tea.KeyCtrlP:
			return m, m.previewVoice()
		case tea.KeyCtrlS:
			return m, m.setDefault()
		case tea.KeyCtrlD:
			return m, m.copyID()
```

- [ ] **Step 2: Update the help bar to show Ctrl alternatives**

Update the help bar in `View()`:

```go
	b.WriteString(helpStyle.Render("  [P/^P] Preview  [S/^S] Set default  [C/^D] Copy ID  [Enter] Select  [Esc] Quit"))
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/pick.go
git commit -m "feat: add Ctrl-key shortcuts for pick actions"
```

---

### Task 5: Write tests for pick model logic

**Files:**
- Create: `cmd/pick_test.go`

- [ ] **Step 1: Write tests for the pick model**

Create `cmd/pick_test.go`:

```go
package cmd

import (
	"testing"

	"github.com/neosapience/cast/internal/client"
)

func testPickVoices() []client.Voice {
	return []client.Voice{
		{VoiceID: "v1", VoiceName: "수진", Gender: "female", Age: "young_adult", UseCases: []string{"Conversational", "News"}},
		{VoiceID: "v2", VoiceName: "건석", Gender: "male", Age: "middle_age", UseCases: []string{"Documentary"}},
		{VoiceID: "v3", VoiceName: "수아", Gender: "female", Age: "teenager", UseCases: []string{"Anime"}},
	}
}

func TestPickModelFilter(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	// Initially all voices visible.
	if len(m.filtered) != 3 {
		t.Fatalf("initial filtered = %d, want 3", len(m.filtered))
	}

	// Simulate typing "수" into the filter.
	m.textInput.SetValue("수")
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Fatalf("after '수' filtered = %d, want 2", len(m.filtered))
	}
	if m.filtered[0].VoiceID != "v1" || m.filtered[1].VoiceID != "v3" {
		t.Errorf("got %v %v, want v1, v3", m.filtered[0].VoiceID, m.filtered[1].VoiceID)
	}
}

func TestPickModelFilterNoMatch(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("없는이름")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Fatalf("filtered = %d, want 0", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestPickModelCursorBounds(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	// Cursor starts at 0.
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Filter down to 1 item, cursor at 2 should clamp.
	m.cursor = 2
	m.textInput.SetValue("건석")
	m.applyFilter()

	if m.cursor != 0 {
		t.Errorf("cursor after filter = %d, want 0", m.cursor)
	}
}

func TestPickModelClearFilter(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("수")
	m.applyFilter()
	if len(m.filtered) != 2 {
		t.Fatalf("filtered = %d, want 2", len(m.filtered))
	}

	// Clear filter — all voices should return.
	m.textInput.SetValue("")
	m.applyFilter()
	if len(m.filtered) != 3 {
		t.Fatalf("after clear filtered = %d, want 3", len(m.filtered))
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./cmd/ -run TestPickModel -v
```

Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/pick_test.go
git commit -m "test: add unit tests for pick model filtering and cursor"
```

---

### Task 6: Manual integration test and final verification

- [ ] **Step 1: Build the binary**

```bash
go build -o cast .
```

- [ ] **Step 2: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests PASS

- [ ] **Step 3: Verify the command registers correctly**

```bash
./cast voices pick --help
```

Expected: shows help with flags `--text`, `--model`, `--gender`, `--age`, `--use-case`

- [ ] **Step 4: Test interactively (requires API key)**

```bash
./cast voices pick --gender female
```

Expected: interactive TUI launches, shows female voices, filter/navigate/shortcuts work

- [ ] **Step 5: Commit any remaining fixes, then clean up build artifact**

```bash
rm -f cast
```

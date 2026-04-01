package cmd

import (
	"fmt"
	"sort"
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

var (
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type pickMode int

const (
	modeBrowse pickMode = iota
	modeAction
)

type preset struct {
	model   string
	emotion string
}

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
	height      int
	mode        pickMode
	presets     []preset
	presetIdx   int
}

func newPickModel(voices []client.Voice, c *client.Client, previewText string) pickModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
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
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeBrowse:
			return m.updateBrowse(msg)
		case modeAction:
			return m.updateAction(msg)
		}
	}

	return m, nil
}

func (m pickModel) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.mode = modeAction
			m.status = ""
			m.presets = buildPresets(m.filtered[m.cursor])
			m.presetIdx = 0
		}
		return m, nil
	}

	// All other keys go to filter input.
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m pickModel) updateAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEsc:
		m.mode = modeBrowse
		m.status = ""
		return m, nil

	case tea.KeyEnter:
		v := m.filtered[m.cursor]
		m.selected = &v
		return m, tea.Quit

	case tea.KeyLeft:
		if len(m.presets) > 0 {
			m.presetIdx = (m.presetIdx - 1 + len(m.presets)) % len(m.presets)
		}
		return m, nil

	case tea.KeyRight:
		if len(m.presets) > 0 {
			m.presetIdx = (m.presetIdx + 1) % len(m.presets)
		}
		return m, nil

	case tea.KeyRunes:
		switch strings.ToLower(string(msg.Runes)) {
		case "p":
			return m, m.previewVoice()
		case "e":
			return m, m.previewSmart()
		case "s":
			return m, m.setDefault()
		case "c":
			return m, m.copyID()
		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func buildPresets(v client.Voice) []preset {
	// Sort models descending so newest version comes first.
	models := make([]client.VoiceModel, len(v.Models))
	copy(models, v.Models)
	sort.Slice(models, func(i, j int) bool {
		return models[i].Version > models[j].Version
	})
	var presets []preset
	for _, m := range models {
		for _, e := range m.Emotions {
			presets = append(presets, preset{model: m.Version, emotion: e})
		}
	}
	return presets
}


func voiceMatchesQuery(v client.Voice, query string) bool {
	for _, field := range []string{
		v.VoiceID,
		v.VoiceName,
		v.Gender,
		v.Age,
		strings.Join(v.UseCases, " "),
	} {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (m *pickModel) applyFilter() {
	query := strings.ToLower(m.textInput.Value())
	if query == "" {
		m.filtered = m.voices
	} else {
		var result []client.Voice
		for _, v := range m.voices {
			if voiceMatchesQuery(v, query) {
				result = append(result, v)
			}
		}
		m.filtered = result
	}
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) == 0 {
			m.cursor = 0
		} else {
			m.cursor = len(m.filtered) - 1
		}
	}
}

func (m pickModel) View() string {
	var b strings.Builder

	// Voice list.
	maxVisible := m.height - 7
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

	listLines := 0
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
		listLines++
	}

	// Pad to push bottom section to the bottom.
	// Bottom takes 4 lines: blank + info + help + trailing newline.
	bottomLines := 4
	totalUsed := listLines + bottomLines
	if totalUsed < m.height {
		for i := 0; i < m.height-totalUsed; i++ {
			b.WriteString("\n")
		}
	}

	// Bottom section: filter or action menu.
	b.WriteString("\n")
	if m.mode == modeBrowse {
		b.WriteString(filterStyle.Render("> "))
		b.WriteString(m.textInput.View())
		b.WriteString("  ")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d/%d", len(m.filtered), len(m.voices))))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  [Enter] Select  [Esc] Quit"))
	} else {
		v := m.filtered[m.cursor]
		b.WriteString(filterStyle.Render(fmt.Sprintf("  %s (%s)", v.VoiceName, v.VoiceID)))
		if len(m.presets) > 0 {
			p := m.presets[m.presetIdx]
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("◀ %s/%s ▶ %d/%d", p.model, p.emotion, m.presetIdx+1, len(m.presets))))
		}
		if m.status != "" {
			b.WriteString("  ")
			b.WriteString(statusStyle.Render(m.status))
		}
		b.WriteString("\n")
		help := "  [P] Preview  [S] Set default  [C] Copy ID  [Enter] Confirm  [Esc] Back"
		if len(m.presets) > 0 && m.presets[m.presetIdx].model == "ssfm-v30" {
			help = "  [P] Preview  [E] Smart  [S] Set default  [C] Copy ID  [Enter] Confirm  [Esc] Back"
		}
		b.WriteString(helpStyle.Render(help))
	}
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
		}
		if len(m.presets) > 0 {
			p := m.presets[m.presetIdx]
			req.Model = p.model
			req.Prompt = &client.TTSPrompt{
				EmotionType:   "preset",
				EmotionPreset: p.emotion,
			}
		} else if len(v.Models) > 0 {
			req.Model = v.Models[0].Version
		}
		audio, err := m.client.TextToSpeech(req)
		if err != nil {
			return statusMsg(fmt.Sprintf("preview failed: %v", err))
		}
		if err := playAudio(audio, "wav"); err != nil {
			return statusMsg(fmt.Sprintf("playback failed: %v", err))
		}
		label := v.VoiceName
		if len(m.presets) > 0 {
			p := m.presets[m.presetIdx]
			label = fmt.Sprintf("%s (%s/%s)", v.VoiceName, p.model, p.emotion)
		}
		return statusMsg(fmt.Sprintf("previewed: %s", label))
	}
}

func (m pickModel) previewSmart() tea.Cmd {
	return func() tea.Msg {
		if len(m.filtered) == 0 {
			return statusMsg("no voice to preview")
		}
		v := m.filtered[m.cursor]
		if len(m.presets) == 0 || m.presets[m.presetIdx].model != "ssfm-v30" {
			return nil
		}
		req := client.TTSRequest{
			VoiceID: v.VoiceID,
			Text:    m.previewText,
			Model:   "ssfm-v30",
			Prompt: &client.TTSPrompt{
				EmotionType: "smart",
			},
		}
		audio, err := m.client.TextToSpeech(req)
		if err != nil {
			return statusMsg(fmt.Sprintf("smart preview failed: %v", err))
		}
		if err := playAudio(audio, "wav"); err != nil {
			return statusMsg(fmt.Sprintf("playback failed: %v", err))
		}
		return statusMsg(fmt.Sprintf("smart previewed: %s", v.VoiceName))
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

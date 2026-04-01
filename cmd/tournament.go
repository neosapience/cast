package cmd

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/neosapience/cast/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tournamentCmd = &cobra.Command{
	Use:   "tournament",
	Short: "Pick a voice through head-to-head elimination rounds",
	RunE: func(cmd *cobra.Command, args []string) error {
		previewText, _ := cmd.Flags().GetString("text")
		if previewText == "" {
			previewText = defaultPreviewText
		}
		size, _ := cmd.Flags().GetInt("size")

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

		if len(voices) < 2 {
			return fmt.Errorf("need at least 2 voices, got %d", len(voices))
		}

		// Shuffle and pick up to size.
		rand.Shuffle(len(voices), func(i, j int) {
			voices[i], voices[j] = voices[j], voices[i]
		})
		// Round down to nearest power of 2.
		if size > len(voices) {
			size = len(voices)
		}
		size = nearestPowerOf2(size)
		pool := voices[:size]

		m := newTournamentModel(pool, c, previewText)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm := finalModel.(tournamentModel)
		if fm.winner != nil {
			fmt.Println(fm.winner.VoiceID)
			if fm.runnerUp != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "runner-up:", fm.runnerUp.VoiceID)
			}
		}
		return nil
	},
}

func nearestPowerOf2(n int) int {
	if n < 2 {
		return 2
	}
	p := int(math.Log2(float64(n)))
	return 1 << p
}

var (
	tournamentTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	tournamentVoiceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	tournamentPickStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	tournamentDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	tournamentHelpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tournamentWinStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true)
	tournamentStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
)

type tournamentModel struct {
	pool        []client.Voice
	round       int
	totalRounds int
	matchIdx    int
	matchesLen  int
	winners     []client.Voice
	winner      *client.Voice
	runnerUp    *client.Voice
	client      *client.Client
	previewText string
	status      string
	height      int
}

func newTournamentModel(pool []client.Voice, c *client.Client, previewText string) tournamentModel {
	totalRounds := int(math.Log2(float64(len(pool))))
	return tournamentModel{
		pool:        pool,
		round:       1,
		totalRounds: totalRounds,
		matchIdx:    0,
		matchesLen:  len(pool) / 2,
		client:      c,
		previewText: previewText,
	}
}

func (m tournamentModel) Init() tea.Cmd {
	return nil
}

func (m tournamentModel) left() client.Voice {
	return m.pool[m.matchIdx*2]
}

func (m tournamentModel) right() client.Voice {
	return m.pool[m.matchIdx*2+1]
}

func (m tournamentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case advanceMsg:
		// Track runner-up from final match.
		if len(m.pool) == 2 {
			loser := m.left()
			if msg.winner.VoiceID == loser.VoiceID {
				loser = m.right()
			}
			m.runnerUp = &loser
		}
		m.winners = append(m.winners, msg.winner)
		m.status = ""
		m.matchIdx++
		if m.matchIdx >= m.matchesLen {
			// Round complete — advance.
			if len(m.winners) == 1 {
				m.winner = &m.winners[0]
				return m, tea.Quit
			}
			m.pool = m.winners
			m.winners = nil
			m.round++
			m.matchIdx = 0
			m.matchesLen = len(m.pool) / 2
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "1":
				return m, m.pick(m.left())
			case "2":
				return m, m.pick(m.right())
			case "p":
				return m, m.preview(m.left())
			case "q":
				return m, m.preview(m.right())
			}
		}
	}

	return m, nil
}

type advanceMsg struct {
	winner client.Voice
}

func (m tournamentModel) pick(v client.Voice) tea.Cmd {
	return func() tea.Msg {
		return advanceMsg{winner: v}
	}
}

func (m tournamentModel) preview(v client.Voice) tea.Cmd {
	return func() tea.Msg {
		model := defaultModel
		if len(v.Models) > 0 {
			model = v.Models[0].Version
		}
		req := client.TTSRequest{
			VoiceID: v.VoiceID,
			Text:    m.previewText,
			Model:   model,
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

func (m tournamentModel) View() string {
	// Winner screen.
	if m.winner != nil {
		var b strings.Builder
		b.WriteString(tournamentWinStyle.Render("  1st"))
		b.WriteString("  ")
		b.WriteString(tournamentPickStyle.Render(fmt.Sprintf("%s (%s)", m.winner.VoiceName, m.winner.VoiceID)))
		b.WriteString("  ")
		b.WriteString(tournamentDimStyle.Render(fmt.Sprintf("%s  %s  %s", m.winner.Gender, m.winner.Age, strings.Join(m.winner.UseCases, ","))))
		b.WriteString("\n")
		if m.runnerUp != nil {
			b.WriteString(tournamentDimStyle.Render("  2nd"))
			b.WriteString("  ")
			b.WriteString(tournamentVoiceStyle.Render(fmt.Sprintf("%s (%s)", m.runnerUp.VoiceName, m.runnerUp.VoiceID)))
			b.WriteString("  ")
			b.WriteString(tournamentDimStyle.Render(fmt.Sprintf("%s  %s  %s", m.runnerUp.Gender, m.runnerUp.Age, strings.Join(m.runnerUp.UseCases, ","))))
			b.WriteString("\n")
		}
		return b.String()
	}

	var b strings.Builder

	// Title.
	stageName := tournamentStageName(len(m.pool))
	b.WriteString(tournamentTitleStyle.Render(fmt.Sprintf("  %s  Match %d/%d", stageName, m.matchIdx+1, m.matchesLen)))
	b.WriteString("\n\n")

	// Voice 1.
	v1 := m.left()
	useCases1 := strings.Join(v1.UseCases, ",")
	b.WriteString(tournamentPickStyle.Render("  1. "))
	b.WriteString(tournamentVoiceStyle.Render(fmt.Sprintf("%-12s %-8s %-14s %s", v1.VoiceName, v1.Gender, v1.Age, useCases1)))
	b.WriteString("\n")

	// Voice 2.
	v2 := m.right()
	useCases2 := strings.Join(v2.UseCases, ",")
	b.WriteString(tournamentPickStyle.Render("  2. "))
	b.WriteString(tournamentVoiceStyle.Render(fmt.Sprintf("%-12s %-8s %-14s %s", v2.VoiceName, v2.Gender, v2.Age, useCases2)))
	b.WriteString("\n")

	// Padding.
	// Top: title(1) + blank(1) + voice1(1) + voice2(1) = 4
	// Bottom: status(1) + help(1) + trailing newline(1) = 3
	contentLines := 7
	if m.height > contentLines {
		for i := 0; i < m.height-contentLines; i++ {
			b.WriteString("\n")
		}
	}

	// Status.
	if m.status != "" {
		b.WriteString(tournamentStatusStyle.Render("  " + m.status))
	}
	b.WriteString("\n")

	// Help.
	b.WriteString(tournamentHelpStyle.Render("  [1] Pick left  [2] Pick right  [P] Preview 1  [Q] Preview 2  [Esc] Quit"))
	b.WriteString("\n")

	return b.String()
}

func tournamentStageName(poolSize int) string {
	switch poolSize {
	case 2:
		return "Final"
	case 4:
		return "Semi-Final"
	default:
		return fmt.Sprintf("Round of %d", poolSize)
	}
}

func init() {
	tournamentCmd.Flags().String("text", "", "Custom text for voice preview")
	tournamentCmd.Flags().Int("size", 8, "Tournament size (rounded down to power of 2)")
	tournamentCmd.Flags().String("model", "", "Filter by model (ssfm-v30, ssfm-v21)")
	tournamentCmd.Flags().String("gender", "", "Filter by gender (male, female)")
	tournamentCmd.Flags().String("age", "", "Filter by age (child, teenager, young_adult, middle_age, elder)")
	tournamentCmd.Flags().String("use-case", "", "Filter by use case")
}

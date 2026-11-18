package theme

import "charm.land/lipgloss/v2"

// Palette is the Nightshift token set from DESIGN.md. It is the only place
// literal color values live.
type Palette struct {
	Base        string
	Surface     string
	Raised      string
	Border      string
	BorderFocus string
	BorderModal string
	Text        string
	Muted       string
	Accent      string
	Accent2     string
	Success     string
	Warning     string
	Error       string
	Info        string
	Match       string
	Selection   string
	ReadOnly    string
	ReadWrite   string
}

var Nightshift = Palette{
	Base:        "#04060A",
	Surface:     "#0A0F15",
	Raised:      "#121A23",
	Border:      "#26333F",
	BorderFocus: "#3EE0D0",
	BorderModal: "#F5B85C",
	Text:        "#F0F6FA",
	Muted:       "#94A6B8",
	Accent:      "#3EE0D0",
	Accent2:     "#B49BFF",
	Success:     "#58E6A0",
	Warning:     "#F5C65F",
	Error:       "#FF6B7D",
	Info:        "#6DB5FF",
	Match:       "#FFD166",
	Selection:   "#0F3A44",
	ReadOnly:    "#7CC4FF",
	ReadWrite:   "#FF9D4D",
}

// Styles are the Lip Gloss projections of the Nightshift roles.
type Styles struct {
	Text    lipgloss.Style
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Accent2 lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	Title       lipgloss.Style
	Header      lipgloss.Style
	Label       lipgloss.Style
	Match       lipgloss.Style
	Select      lipgloss.Style
	Cursor      lipgloss.Style
	ModeRO      lipgloss.Style
	ModeRW      lipgloss.Style
	Border      lipgloss.Style
	Focused     lipgloss.Style
	Surface     lipgloss.Style
	Raised      lipgloss.Style
	BorderModal lipgloss.Style
}

func New() Styles {
	return Styles{
		Text:    lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Text)),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Muted)),
		Accent:  lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Accent)),
		Accent2: lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Accent2)),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Success)),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Warning)),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Error)),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Info)),

		Title:       lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Text)).Bold(true),
		Header:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Muted)).Bold(true),
		Label:       lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Muted)).Bold(true),
		Match:       lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Base)).Background(lipgloss.Color(Nightshift.Match)),
		Select:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Text)).Background(lipgloss.Color(Nightshift.Selection)).Reverse(true).Bold(true),
		Cursor:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Base)).Background(lipgloss.Color(Nightshift.Accent)).Bold(true),
		ModeRO:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.ReadOnly)).Bold(true),
		ModeRW:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.ReadWrite)).Bold(true),
		Border:      lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.Border)),
		Focused:     lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.BorderFocus)).Bold(true),
		Surface:     lipgloss.NewStyle().Background(lipgloss.Color(Nightshift.Surface)),
		Raised:      lipgloss.NewStyle().Background(lipgloss.Color(Nightshift.Raised)),
		BorderModal: lipgloss.NewStyle().Foreground(lipgloss.Color(Nightshift.BorderModal)),
	}
}

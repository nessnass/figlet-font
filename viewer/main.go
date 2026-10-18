// nessnass — a terminal viewer for the Nessnass figlet font.
//
// The rendered font is the edit surface: type and it re-renders live. It reads
// Nessnass.flf with its own renderer (flf.go), so figlet isn't needed.
//
// Usage, from this folder:
//
//	go run .              start with "Nessnass"
//	go run . "my text"    start with your own text
//
// Keys:
//
//	type / backspace   edit the text      enter          new line
//	option+space       half space
//	← → ↑ ↓ home end   move the cursor    ctrl+a         show all characters
//	tab / shift+tab    colour             ctrl+l         clear
//	ctrl+w             save as .txt       esc / ctrl+c   quit
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	fontName       = "Nessnass"
	colorAccent    = "#5599cc"
	headerOverhead = 5 // blank, title, info, statusLine (even when empty), help

	// The font's half space, one column wide: an en space (U+2002)
	halfSpace = ' '
)

// ============================================================================
// figlet + colours
// ============================================================================

// findFontFile locates Nessnass.flf, checking source-, executable- and
// cwd-relative locations so it works under `go run .`, a built binary, or from
// the parent folder.
func findFontFile() string {
	var dirs []string
	if _, file, _, ok := runtime.Caller(0); ok {
		dirs = append(dirs, filepath.Dir(file), filepath.Dir(filepath.Dir(file)))
	}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		dirs = append(dirs, d, filepath.Dir(d))
	}
	dirs = append(dirs, ".", "..")
	for _, d := range dirs {
		p := filepath.Join(d, fontName+".flf")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

type colorMode struct {
	name   string
	repHex string // representative hex for UI labels
	fgEsc  string // ANSI escape to restore foreground, empty = don't override
	apply  func(string) string
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 0)
	g, _ := strconv.ParseInt(hex[2:4], 16, 0)
	b, _ := strconv.ParseInt(hex[4:6], 16, 0)
	return int(r), int(g), int(b)
}

func solid(name, hex string) colorMode {
	r, g, b := hexToRGB(hex)
	fgEsc := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	return colorMode{name, hex, fgEsc, func(s string) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true).Render(s)
	}}
}

// The main colours of the first eight Nessnass artworks, in token order
// (https://www.nessnass.com)
func palettes() []colorMode {
	return []colorMode{
		solid("sky blue", "#87ceeb"),
		solid("pink", "#ffc0cb"),
		solid("green", "#00ff00"),
		solid("indigo", "#4b0082"),
		solid("light green", "#90ee90"),
		solid("yellow", "#ffff00"),
		solid("gold", "#ffd700"),
		solid("red", "#ff0000"),
	}
}

// ============================================================================
// model
// ============================================================================

type model struct {
	colors    []colorMode
	colorIdx  int
	word      string
	font      *figFont
	preview   string
	status    string
	width     int
	height    int
	cursorPos int
	spacedAll bool
	quitting  bool
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) refresh() {
	if m.word == "" {
		m.preview = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).
			Render("(type to render…)")
		return
	}
	lines := strings.Split(m.word, "\n")
	globalPos := 0
	var blocks []string
	for _, line := range lines {
		if line == "" {
			blocks = append(blocks, "")
			globalPos++
			continue
		}
		lineLen := len([]rune(line))
		cursorHere := m.cursorPos >= globalPos && m.cursorPos <= globalPos+lineLen
		var rendered string
		if cursorHere {
			rendered = m.colors[m.colorIdx].apply(m.font.renderWithCursor(line, m.cursorPos-globalPos, m.colors[m.colorIdx].fgEsc))
		} else {
			rendered = m.colors[m.colorIdx].apply(m.font.render(line))
		}
		blocks = append(blocks, rendered)
		globalPos += lineLen + 1
	}
	m.preview = strings.Join(blocks, "\n")
}

// export writes the current rendering to a plain .txt file (no colour codes).
func (m *model) export() {
	if m.word == "" {
		return
	}
	var blocks []string
	for _, line := range strings.Split(m.word, "\n") {
		if line == "" {
			blocks = append(blocks, "")
			continue
		}
		blocks = append(blocks, m.font.render(line))
	}
	art := strings.Join(blocks, "\n")
	name := sanitizeName(m.word) + "_" + fontName + ".txt"
	if err := os.WriteFile(name, []byte(art+"\n"), 0644); err != nil {
		m.status = "export failed: " + err.Error()
		return
	}
	abs, _ := filepath.Abs(name)
	m.status = "saved → " + abs
}

func sanitizeName(s string) string {
	repl := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}
	out := strings.Trim(strings.Map(repl, s), "-")
	if out == "" {
		out = "art"
	}
	return out
}

func (m model) moveLine(dir int) int {
	runes := []rune(m.word)
	if len(runes) == 0 {
		return 0
	}
	lineStarts := []int{0}
	for i, r := range runes {
		if r == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	lineEnds := make([]int, len(lineStarts))
	for i := range lineStarts {
		if i+1 < len(lineStarts) {
			lineEnds[i] = lineStarts[i+1] - 1
		} else {
			lineEnds[i] = len(runes)
		}
	}
	curLine := 0
	for i := range lineStarts {
		if m.cursorPos >= lineStarts[i] && (i+1 >= len(lineStarts) || m.cursorPos <= lineEnds[i]) {
			curLine = i
			break
		}
	}
	target := curLine + dir
	if target < 0 || target >= len(lineStarts) {
		return m.cursorPos
	}
	col := m.cursorPos - lineStarts[curLine]
	targetPos := lineStarts[target] + col
	if targetPos > lineEnds[target] {
		targetPos = lineEnds[target]
	}
	return targetPos
}

func allCharsString(width int, spaced bool) string {
	groups := []string{
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"abcdefghijklmnopqrstuvwxyz",
		"0123456789",
		"!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~",
	}

	avgGlyphWidth := 4
	if spaced {
		avgGlyphWidth = 5
	}
	avail := width - 4
	if avail < avgGlyphWidth {
		avail = avgGlyphWidth
	}
	perRow := avail / avgGlyphWidth
	if perRow < 3 {
		perRow = 3
	}

	join := func(rs []rune) string {
		if !spaced {
			return string(rs)
		}
		var b strings.Builder
		for i, r := range rs {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	var b strings.Builder
	for gi, group := range groups {
		if gi > 0 {
			b.WriteByte('\n')
		}
		rs := []rune(group)
		for i := 0; i < len(rs); i += perRow {
			if i > 0 {
				b.WriteByte('\n')
			}
			end := i + perRow
			if end > len(rs) {
				end = len(rs)
			}
			b.WriteString(join(rs[i:end]))
		}
	}
	return b.String()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.refresh()
		return m, nil

	case tea.KeyMsg:
		m.status = ""

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyTab:
			m.colorIdx = (m.colorIdx + 1) % len(m.colors)
			m.refresh()
			return m, nil
		case tea.KeyShiftTab:
			m.colorIdx = (m.colorIdx - 1 + len(m.colors)) % len(m.colors)
			m.refresh()
			return m, nil

		case tea.KeyCtrlW:
			m.export()
			return m, nil
		case tea.KeyCtrlA:
			m.word = allCharsString(m.width, m.spacedAll)
			m.cursorPos = len([]rune(m.word))
			m.spacedAll = !m.spacedAll
			m.refresh()
			return m, nil
		case tea.KeyCtrlL:
			m.word = ""
			m.cursorPos = 0
			m.spacedAll = false
			m.refresh()
			return m, nil

		case tea.KeyLeft:
			if m.cursorPos > 0 {
				m.cursorPos--
			}
			m.refresh()
			return m, nil
		case tea.KeyRight:
			r := []rune(m.word)
			if m.cursorPos < len(r) {
				m.cursorPos++
			}
			m.refresh()
			return m, nil
		case tea.KeyUp:
			m.cursorPos = m.moveLine(-1)
			m.refresh()
			return m, nil
		case tea.KeyDown:
			m.cursorPos = m.moveLine(1)
			m.refresh()
			return m, nil
		case tea.KeyHome:
			m.cursorPos = 0
			m.refresh()
			return m, nil
		case tea.KeyEnd:
			m.cursorPos = len([]rune(m.word))
			m.refresh()
			return m, nil

		case tea.KeyBackspace:
			if m.cursorPos > 0 {
				r := []rune(m.word)
				m.word = string(append(r[:m.cursorPos-1], r[m.cursorPos:]...))
				m.cursorPos--
			}
			m.refresh()
			return m, nil

		case tea.KeySpace:
			ch := ' '
			if msg.Alt {
				ch = halfSpace
			}
			r := []rune(m.word)
			var buf []rune
			buf = append(buf, r[:m.cursorPos]...)
			buf = append(buf, ch)
			buf = append(buf, r[m.cursorPos:]...)
			m.word = string(buf)
			m.cursorPos++
			m.refresh()
			return m, nil

		case tea.KeyEnter:
			r := []rune(m.word)
			var buf []rune
			buf = append(buf, r[:m.cursorPos]...)
			buf = append(buf, '\n')
			buf = append(buf, r[m.cursorPos:]...)
			m.word = string(buf)
			m.cursorPos++
			m.refresh()
			return m, nil

		case tea.KeyRunes:
			// macOS types option+space as a no-break space: a half space here
			runes := make([]rune, len(msg.Runes))
			for i, ch := range msg.Runes {
				if ch == ' ' {
					ch = halfSpace
				}
				runes[i] = ch
			}
			r := []rune(m.word)
			var buf []rune
			buf = append(buf, r[:m.cursorPos]...)
			buf = append(buf, runes...)
			buf = append(buf, r[m.cursorPos:]...)
			m.word = string(buf)
			m.cursorPos += len(msg.Runes)
			m.refresh()
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	title := lipgloss.NewStyle().
		Background(lipgloss.Color(colorAccent)).Foreground(lipgloss.Color("#ffffff")).
		Bold(true).Padding(0, 1).
		Render("Nessnass — figlet font viewer")

	info := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render("  colour: ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(m.colors[m.colorIdx].repHex)).Render(m.colors[m.colorIdx].name)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777")).Render(
		"  type · opt+space half space · ←→↑↓ home end · enter br · tab/shift+tab colour · ctrl+a all · ctrl+l clear · ctrl+w export · esc quit")

	statusLine := ""
	if m.status != "" {
		statusLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50")).Bold(true).
			Render("  " + m.status)
	}

	previewRows := m.height - headerOverhead
	if previewRows < 3 {
		previewRows = 3
	}
	preview := padMin(indent(m.preview), previewRows)

	return strings.Join([]string{
		"", title, info, statusLine,
		preview,
		help,
	}, "\n")
}

func indent(s string) string { return "  " + strings.ReplaceAll(s, "\n", "\n  ") }

// padMin pads the block to at least minRows so the layout doesn't jump when the
// text is short — but never truncates, so multi-line output can grow freely.
func padMin(s string, minRows int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < minRows {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// ============================================================================
// main
// ============================================================================

func main() {
	word := "Nessnass"
	if len(os.Args) > 1 {
		word = strings.Join(os.Args[1:], " ")
	}

	path := findFontFile()
	if path == "" {
		fmt.Fprintln(os.Stderr, "could not find "+fontName+".flf (expected in this folder or its parent)")
		os.Exit(1)
	}
	font, err := loadFigFont(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not load "+fontName+".flf:", err)
		os.Exit(1)
	}

	m := model{
		colors: palettes(),
		word:   word,
		font:   font,
	}

	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

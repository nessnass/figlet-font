// A tiny, self-contained figlet font (.flf) parser + renderer — just enough to draw
// the Nessnass font, so the tool needs no external `figlet` binary.
//
// It does full-width horizontal layout (no smushing), which is exactly how the
// Nessnass glyphs are designed to abut. Multi-line input is handled by the
// caller (one render per line).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type figFont struct {
	height int
	glyphs map[rune][]string // rune → height lines, hardblanks already spaces
}

// loadFigFont parses a .flf file. It reads the 95 required ASCII glyphs
// (32–126), the 7 German glyphs, and any trailing code-tagged glyphs.
func loadFigFont(path string) (*figFont, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	if !sc.Scan() {
		return nil, fmt.Errorf("empty font file")
	}
	header := sc.Text()
	if !strings.HasPrefix(header, "flf2a") || len(header) < 6 {
		return nil, fmt.Errorf("not a flf2a figlet font")
	}
	hardblank := rune(header[5])
	fields := strings.Fields(header[6:])
	if len(fields) < 5 {
		return nil, fmt.Errorf("bad figlet font header")
	}
	height, _ := strconv.Atoi(fields[0])
	comments, _ := strconv.Atoi(fields[4])
	if height < 1 {
		return nil, fmt.Errorf("bad height %d", height)
	}

	// Skip the comment block.
	for i := 0; i < comments; i++ {
		if !sc.Scan() {
			return nil, fmt.Errorf("unexpected EOF in comments")
		}
	}

	ff := &figFont{height: height, glyphs: map[rune][]string{}}

	readGlyph := func() ([]string, bool) {
		rows := make([]string, 0, height)
		for i := 0; i < height; i++ {
			if !sc.Scan() {
				return nil, false
			}
			rows = append(rows, cleanRow(sc.Text(), hardblank))
		}
		return padBlock(rows), true
	}

	// Required ASCII glyphs 32..126, then the 7 German glyphs.
	codes := []rune{}
	for c := rune(32); c <= 126; c++ {
		codes = append(codes, c)
	}
	codes = append(codes, 196, 214, 220, 228, 246, 252, 223) // Ä Ö Ü ä ö ü ß
	for _, c := range codes {
		g, ok := readGlyph()
		if !ok {
			return nil, fmt.Errorf("unexpected EOF reading glyph %d", c)
		}
		ff.glyphs[c] = g
	}

	// Trailing code-tagged glyphs: "<code> [comment]" then height rows.
	for sc.Scan() {
		tag := strings.TrimSpace(sc.Text())
		if tag == "" {
			continue
		}
		code, err := parseCode(strings.Fields(tag)[0])
		if err != nil {
			continue // not a code tag — ignore
		}
		g, ok := readGlyph()
		if !ok {
			break
		}
		ff.glyphs[code] = g
	}

	return ff, nil
}

// cleanRow strips the trailing endmark run and turns hardblanks into spaces.
func cleanRow(line string, hardblank rune) string {
	r := []rune(line)
	if len(r) > 0 {
		end := r[len(r)-1]
		for len(r) > 0 && r[len(r)-1] == end {
			r = r[:len(r)-1]
		}
	}
	for i, c := range r {
		if c == hardblank {
			r[i] = ' '
		}
	}
	return string(r)
}

// padBlock right-pads every row of a glyph to the glyph's widest row, so the
// columns line up when glyphs are concatenated.
func padBlock(rows []string) []string {
	w := 0
	for _, r := range rows {
		if n := len([]rune(r)); n > w {
			w = n
		}
	}
	for i, r := range rows {
		if pad := w - len([]rune(r)); pad > 0 {
			rows[i] = r + strings.Repeat(" ", pad)
		}
	}
	return rows
}

func parseCode(s string) (rune, error) {
	// figlet code tags: decimal, 0x.. hex, or 0.. octal.
	base := 10
	neg := false
	t := s
	if strings.HasPrefix(t, "-") {
		neg, t = true, t[1:]
	}
	switch {
	case strings.HasPrefix(t, "0x"), strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	v, err := strconv.ParseInt(t, base, 32)
	if err != nil {
		return 0, err
	}
	if neg {
		v = -v
	}
	return rune(v), nil
}

// render lays out a single line of text in the font, full-width. Unknown runes
// are skipped.
func (ff *figFont) render(text string) string {
	rows := make([]string, ff.height)
	for _, ch := range text {
		g, ok := ff.glyphs[ch]
		if !ok {
			continue
		}
		for i := 0; i < ff.height && i < len(g); i++ {
			rows[i] += g[i]
		}
	}
	return strings.Join(rows, "\n")
}

// renderWithCursor renders a single line with a white-background cursor at the
// given local position (0-based within the line). Cursor at lineLen shows a
// trailing block. fgEsc is an optional ANSI escape to restore the foreground
// after the cursor so subsequent characters keep their colour.
func (ff *figFont) renderWithCursor(text string, cursorLocal int, fgEsc string) string {
	rows := make([]string, ff.height)
	pos := 0
	for _, ch := range text {
		g, ok := ff.glyphs[ch]
		if !ok {
			pos++
			continue
		}
		for i := 0; i < ff.height && i < len(g); i++ {
			if pos == cursorLocal && fgEsc != "" {
				rows[i] += "\x1b[38;2;0;0;0;48;2;255;255;255m" + g[i] + "\x1b[49m" + fgEsc
			} else if pos == cursorLocal {
				rows[i] += "\x1b[48;2;255;255;255m" + g[i] + "\x1b[49m"
			} else {
				rows[i] += g[i]
			}
		}
		pos++
	}
	if cursorLocal >= pos {
		w := 3
		if sg, ok := ff.glyphs[' ']; ok && len(sg) > 0 {
			w = len([]rune(sg[0]))
		}
		for i := 0; i < ff.height; i++ {
			rows[i] += "\x1b[48;2;255;255;255m" + strings.Repeat(" ", w) + "\x1b[49m"
		}
	}
	return strings.Join(rows, "\n")
}

package tui

import (
	"fmt"
	"strings"
)

func helpBar(keys []string, width int) string {
	var parts []string
	for i := 0; i < len(keys); i += 2 {
		key := keys[i]
		desc := ""
		if i+1 < len(keys) {
			desc = keys[i+1]
		}
		parts = append(parts, helpStyle.Render(key+" "+desc))
	}
	bar := strings.Join(parts, "  ")

	padding := width - len(bar) - 4
	if padding < 0 {
		padding = 0
	}
	return helpStyle.Render(strings.Repeat(" ", padding)) + "\n" + bar
}

// helpOverlay renders a centered overlay with all key bindings for the current view.
// bindings is [][2]string where each entry is {key, description}.
func helpOverlay(title string, bindings [][2]string, width int) string {
	keyW := 0
	for _, b := range bindings {
		if len(b[0]) > keyW {
			keyW = len(b[0])
		}
	}

	var lines []string
	lines = append(lines, titleStyle.Render(" "+title+" "))
	lines = append(lines, "")
	for _, b := range bindings {
		key := fmt.Sprintf("%-*s", keyW, b[0])
		lines = append(lines, fmt.Sprintf("  %s  %s", selectedItemStyle.Render(key), b[1]))
	}
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("press ? or esc to close"))

	content := strings.Join(lines, "\n")
	box := helpOverlayStyle.Render(content)

	// Pad horizontally to center
	lines2 := strings.Split(box, "\n")
	maxW := 0
	for _, l := range lines2 {
		w := lipglossWidth(l)
		if w > maxW {
			maxW = w
		}
	}
	padLeft := (width - maxW) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	pad := strings.Repeat(" ", padLeft)
	var b strings.Builder
	for _, l := range lines2 {
		b.WriteString(pad + l + "\n")
	}
	return b.String()
}

// lipglossWidth counts visible characters (ignoring ANSI escape sequences).
func lipglossWidth(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

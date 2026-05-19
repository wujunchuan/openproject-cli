package tui

import "strings"

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

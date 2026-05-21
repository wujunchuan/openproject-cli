package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const assigneeHighlightsFileName = "tui-assignee-highlights.json"

type AssigneeHighlights struct {
	Highlights map[string]string `json:"highlights"`
}

func assigneeHighlightsFilePath() string {
	return filepath.Join(configFileDir(), assigneeHighlightsFileName)
}

func LoadAssigneeHighlights() (map[string]string, error) {
	var ah AssigneeHighlights
	data, err := os.ReadFile(assigneeHighlightsFilePath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &ah)
	if err != nil {
		return nil, err
	}
	return ah.Highlights, nil
}

func SaveAssigneeHighlights(highlights map[string]string) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	ah := AssigneeHighlights{Highlights: highlights}
	data, err := json.MarshalIndent(ah, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(assigneeHighlightsFilePath(), data, 0644)
}

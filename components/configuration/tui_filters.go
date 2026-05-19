package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const filterFileName = "tui-filters.json"

type FilterState struct {
	Project  string `json:"project"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Assignee string `json:"assignee"`
}

func filterFilePath() string {
	return filepath.Join(configFileDir(), filterFileName)
}

func SaveFilters(filters FilterState) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(filters, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filterFilePath(), data, 0644)
}

func LoadFilters() (FilterState, error) {
	var fs FilterState
	data, err := os.ReadFile(filterFilePath())
	if os.IsNotExist(err) {
		return fs, nil
	}
	if err != nil {
		return fs, err
	}
	err = json.Unmarshal(data, &fs)
	return fs, err
}

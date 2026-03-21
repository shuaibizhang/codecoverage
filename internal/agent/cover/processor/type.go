package processor

import (
	"errors"

	"github.com/shuaibizhang/codecoverage/internal/parser"
)

type ModuleInfo struct {
	Language   parser.LanguageType `json:"language"`
	Module     string              `json:"module"`
	Branch     string              `json:"branch"`
	Commit     string              `json:"commit"`
	BaseCommit string              `json:"base_commit"`
	BuildID    string              `json:"build_id"`
	SessionID  string              `json:"session_id"`
}

func (m *ModuleInfo) IsValid() error {
	if m.Module == "" {
		return errors.New("module is empty")
	}
	return nil
}

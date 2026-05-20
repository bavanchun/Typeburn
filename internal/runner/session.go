// Package runner builds typing sessions shared by the TUI and CLI paths.
package runner

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/typing"
	"github.com/bavanchun/Typeburn/internal/words"
)

// Session is the pure typing state needed to run a test.
type Session struct {
	Engine   *typing.Engine
	Target   string
	Mode     config.Mode
	Length   int
	QuoteLen words.QuoteLen
	CodeText string
}

// NewSession builds a fresh non-code typing session.
// seed==0 uses the words package's time-based random seed.
func NewSession(mode config.Mode, length int, ql words.QuoteLen, seed int64) Session {
	g := words.NewGenerator(seed)
	target := words.ForMode(g, mode, length, ql)
	return Session{
		Engine:   RebuildEngine(target, mode, length),
		Target:   target,
		Mode:     mode,
		Length:   length,
		QuoteLen: ql,
	}
}

// NewCodeSession builds a Code-mode session from an already-normalized snippet.
func NewCodeSession(snippet string) Session {
	return Session{
		Engine:   RebuildEngine(snippet, config.ModeCode, 0),
		Target:   snippet,
		Mode:     config.ModeCode,
		CodeText: snippet,
	}
}

// RebuildEngine returns a fresh engine for an existing target.
func RebuildEngine(target string, mode config.Mode, length int) *typing.Engine {
	return typing.New(target, mode, wordTarget(mode, length))
}

func wordTarget(mode config.Mode, length int) int {
	if mode == config.ModeTime {
		return length * 1000
	}
	return length
}

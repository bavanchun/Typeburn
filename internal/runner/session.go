// Package runner builds typing sessions shared by the TUI and CLI paths.
package runner

import (
	"github.com/bavanchun/Typeburn/internal/mode"
	"github.com/bavanchun/Typeburn/internal/typing"
	"github.com/bavanchun/Typeburn/internal/words"
)

// Session is the pure typing state needed to run a test.
type Session struct {
	Engine   *typing.Engine
	Target   string
	Mode     mode.Mode
	Length   int
	QuoteLen words.QuoteLen
	CodeText string
	Strict   bool
}

// NewSession builds a fresh non-code typing session.
// seed==0 uses the words package's time-based random seed.
func NewSession(m mode.Mode, length int, ql words.QuoteLen, seed int64, strict bool) Session {
	g := words.NewGenerator(seed)
	target := words.ForMode(g, m, length, ql)
	return Session{
		Engine:   RebuildEngine(target, m, length, strict),
		Target:   target,
		Mode:     m,
		Length:   length,
		QuoteLen: ql,
		Strict:   strict,
	}
}

// NewCodeSession builds a Code-mode session from an already-normalized snippet.
func NewCodeSession(snippet string, strict bool) Session {
	return Session{
		Engine:   RebuildEngine(snippet, mode.ModeCode, 0, strict),
		Target:   snippet,
		Mode:     mode.ModeCode,
		CodeText: snippet,
		Strict:   strict,
	}
}

// RebuildEngine returns a fresh engine for an existing target.
func RebuildEngine(target string, m mode.Mode, length int, strict bool) *typing.Engine {
	return typing.NewStrict(target, m, wordTarget(m, length), strict)
}

func wordTarget(m mode.Mode, length int) int {
	if m == mode.ModeTime {
		return length * 1000
	}
	return length
}

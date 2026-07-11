package ui

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

func TestRestartSame_ResetsTimers(t *testing.T) {
	m := NewTyping(config.ModeWords, 10, 0, theme.Default(), config.DefaultKeymap(), false, false, false, false)
	m.w, m.h = 80, 24
	m2 := m.restartSame()
	if m2.startMs != 0 || m2.nowMs != 0 || m2.headerWPM != 0 {
		t.Error("restartSame should zero timing fields")
	}
}

func TestRestartSame_PreservesTarget(t *testing.T) {
	m := NewTyping(config.ModeWords, 10, 0, theme.Default(), config.DefaultKeymap(), false, false, false, false)
	orig := m.TargetText()
	m2 := m.restartSame()
	if m2.TargetText() != orig {
		t.Error("restartSame should preserve target text")
	}
}

func TestNewTest_PreservesSize(t *testing.T) {
	m := NewTyping(config.ModeWords, 10, 0, theme.Default(), config.DefaultKeymap(), false, false, false, false)
	m.w, m.h = 80, 24
	m2 := m.newTest()
	if m2.w != 80 || m2.h != 24 {
		t.Errorf("newTest size = %dx%d, want 80x24", m2.w, m2.h)
	}
}

func TestNewTest_CodePreservesTarget(t *testing.T) {
	snippet := "func main() {}"
	m := NewTypingCode(snippet, theme.Default(), config.DefaultKeymap(), false, false)
	m2 := m.newTest()
	if m2.TargetText() != snippet {
		t.Errorf("code newTest target = %q, want %q", m2.TargetText(), snippet)
	}
}

func TestNewTest_PreservesPunctuationAndNumbers(t *testing.T) {
	m := newTypingWithSeed(
		config.ModeWords, 10, 0, theme.Default(), config.DefaultKeymap(), false, false, true, true, 42,
	)
	m2 := m.newTest()
	if !m2.punctuation || !m2.numbers {
		t.Errorf("newTest (ctrl+r) dropped punctuation/numbers: got punctuation=%v numbers=%v", m2.punctuation, m2.numbers)
	}
}

func TestApplySettings_UpdatesBlink(t *testing.T) {
	m := NewTyping(config.ModeWords, 10, 0, theme.Default(), config.DefaultKeymap(), false, false, false, false)
	s := config.Defaults()
	s.BlinkCursor = true
	m2 := m.ApplySettings(s, theme.Default())
	if !m2.blink {
		t.Error("ApplySettings should update blink")
	}
}

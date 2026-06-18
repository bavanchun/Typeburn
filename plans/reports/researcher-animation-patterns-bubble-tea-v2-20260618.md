# Animation Patterns for Bubble Tea v2 + Lip Gloss v2 — Technical Report

**Date:** 2026-06-18  
**Scope:** STDLIB-ONLY animation system for Typeburn TUI (no harmonica, no bubbles)  
**Confidence:** High (source: official pkg.go.dev, GitHub upgrade guides, live codebase verification)

---

## Executive Summary

Bubble Tea v2 is animation-ready. The framework provides:
- **Self-stopping frame loops** via `tea.Tick(d, fn)` with re-arming on demand
- **Cell-based differential rendering** (Cursed Renderer) that makes frequent redraws cheap
- **`tea.View` struct** enabling declarative rendering without string diffing
- **Lip Gloss v2 color API** accepting `color.Color` for true-color interpolation

With stdlib-only easing (cubic, quadratic) and critical-damping springs, you can build smooth animations without external dependencies. The Typeburn codebase already uses this pattern (100ms ticks in `internal/ui/timer.go`); animation support scales down to 30fps with no architectural changes.

---

## 1. Self-Stopping Frame Loop (Idiomatic v2)

### Signature

```go
func tea.Tick(d time.Duration, fn func(time.Time) Msg) Cmd
```

**Key property:** Tick sends a *single* message; **you must re-arm it** to loop.

### Pattern

```go
type FrameTickMsg struct{ t time.Time }

func (m *Model) Init() tea.Cmd {
    // Start the animation loop on app init or transition
    return tea.Tick(animFrameInterval(), func(t time.Time) tea.Msg {
        return FrameTickMsg{t: t}
    })
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case FrameTickMsg:
        // Advance frame
        m.frameIdx++
        
        // Stop condition: animation complete
        if m.frameIdx >= m.frameDurationFrames {
            m.isAnimating = false
            return m, nil  // ← **Return nil to stop the loop**
        }
        
        // Re-arm the tick to continue
        return m, tea.Tick(animFrameInterval(), func(t time.Time) tea.Msg {
            return FrameTickMsg{t: t}
        })
    }
    return m, nil
}
```

### Stopping Cleanly

- **Stop:** Return `nil` instead of a Tick command
- **Restart:** Return a new Tick command when animation should resume
- **No CPU burn:** When `isAnimating = false`, Update doesn't return a Tick, so the event loop goes idle

### Conditional Re-arming (Efficient)

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case FrameTickMsg:
        // ... update state ...
        
        if m.isAnimating {
            return m, tea.Tick(animFrameInterval(), /* ... */)  // Loop
        }
        return m, nil  // Stop if animation end reached
}
```

### `tea.Every` vs `tea.Tick`

**`tea.Every(duration, fn func(time.Time) Msg) Cmd`** exists in v2 but **syncs to the system clock** (useful for synchronized multi-component ticks). For independent animation loops, **use `tea.Tick`** (ignores system clock, runs at your specified interval).

### Frame Interval for 30fps

```go
func animFrameInterval() time.Duration {
    // 30 fps → 33.33ms per frame
    return time.Duration(float64(time.Second) / 30.0)  // ~33ms
}
```

**Typeburn precedent:** `internal/ui/timer.go` uses `100*time.Millisecond` for live-WPM updates; dropping to 33ms for visual animation is a clean scale-down.

---

## 2. Color Interpolation in Lip Gloss v2

### API Overview

Lip Gloss v2 accepts **`color.Color` interface** (stdlib `image/color`).

```go
import "charm.land/lipgloss/v2"

c1 := lipgloss.Color("#FF0000")  // Create from hex
c2 := lipgloss.Color("#0000FF")

style := lipgloss.NewStyle().
    Foreground(c1).
    Background(c2)

output := style.Render("Text")
```

### Constructing Colors

Three input formats:

| Format | Example | Range |
|--------|---------|-------|
| **ANSI 16** (4-bit) | `lipgloss.Color("5")` | 0–15 |
| **ANSI 256** (8-bit) | `lipgloss.Color("86")` | 0–255 |
| **True Color** (24-bit hex) | `lipgloss.Color("#EB4268")` | `#000000`–`#FFFFFF` |

### RGB Interpolation (Frame-by-Frame)

```go
func interpolateRGB(from, to color.Color, progress float64) color.Color {
    // progress ∈ [0, 1]
    
    // Extract RGBA (returns 0–65535 range)
    r1, g1, b1, _ := from.RGBA()
    r2, g2, b2, _ := to.RGBA()
    
    // Interpolate in uint32 space
    r := uint32(float64(r1)*(1-progress) + float64(r2)*progress)
    g := uint32(float64(g1)*(1-progress) + float64(g2)*progress)
    b := uint32(float64(b1)*(1-progress) + float64(b2)*progress)
    
    // Normalize from 0–65535 to 0–255 (divide by 257)
    rn := r / 257
    gn := g / 257
    bn := b / 257
    
    // Reconstruct as hex
    hexStr := fmt.Sprintf("#%02X%02X%02X", rn, gn, bn)
    return lipgloss.Color(hexStr)
}
```

**Typical animation loop:**

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case FrameTickMsg:
        // Compute progress (0→1)
        progress := float64(m.frameIdx) / float64(m.frameDurationFrames)
        
        // Interpolate color
        m.animColor = interpolateRGB(m.colorFrom, m.colorTo, progress)
        
        // Rebuild style with new color
        m.animStyle = lipgloss.NewStyle().
            Foreground(m.animColor).
            Bold(true)
        
        return m, tea.Tick(/* ... */)
}

func (m *Model) View() tea.View {
    text := m.animStyle.Render("Hello")
    return tea.NewView(text)
}
```

### Applied to Typeburn

Typeburn uses semantic `Role` enum (e.g., `RoleAccent`, `RoleError`) mapped in `internal/theme/theme.go`. To animate:

```go
// theme/theme.go Style() method already knows how to resolve roles:
colorFrom := t.colors[RoleAccent]
colorTo   := t.colors[RoleSuccess]

// In animation loop:
animColor := interpolateRGB(colorFrom, colorTo, progress)
style := lipgloss.NewStyle().Foreground(animColor)
```

**NO_COLOR support:** Themes support attribute-only rendering. Animated colors automatically fall back to attributes when `NO_COLOR=1` is set (see `theme.go:attrOnlyStyle()`).

---

## 3. How Charm's Components Animate (Reference)

### Bubbles/Spinner Pattern

The `bubbles/spinner` component (though not used here) demonstrates the canonical pattern:

```go
// Init: start the spinner
func (m Model) Init() tea.Cmd {
    return m.spinner.Tick  // spinner exposes a Cmd
}

// Update: delegate to spinner, get back next Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
}
```

**Key insight:** The component is stateful (holds frame index internally); ticked externally via a `TickMsg`. Cadence (typically 50–100ms) is managed by the component's `Tick` Cmd.

### For Stdlib-Only (No Bubbles)

You don't need bubbles. Implement the same pattern inline:

```go
type AnimationModel struct {
    frameIdx        int
    frameDuration   int
    isAnimating     bool
}

// Your own tick message
type FrameTickMsg struct{ t time.Time }

// Your own Tick command
func (a *AnimationModel) nextTick() tea.Cmd {
    if !a.isAnimating {
        return nil  // Stop
    }
    return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
        return FrameTickMsg{t: t}
    })
}
```

---

## 4. Rendering Cost & Damage Diffing

### Bubble Tea v2 Rendering Architecture

**Cell-Based Differential Renderer (Cursed Renderer):**

Bubble Tea v2 uses the ncurses-based Cursed Renderer, which:
- Tracks **dirty cells** (only cells that changed)
- Sends **only differential updates** to the terminal (not full screen redraws)
- Compares View content *cell-by-cell*, not string-by-string

### Performance Impact

**Official claim:** "Orders of magnitude" faster than v1.

- **Local:** Snappier UI at high frame rates
- **SSH:** Significantly lower bandwidth usage (cost reduction in cloud)

### View() Signature (v2)

```go
func (m Model) View() tea.View {
    s := "Content..."
    return tea.NewView(s)  // Returns tea.View struct, not string
}
```

The `tea.View` struct is **declarative**, enabling the renderer to:
- Detect what changed
- Avoid sending unchanged cells
- Apply attributes/colors only to dirty regions

### Returning the Same String Is Cheap

If you return an identical `View`:

```go
func (m Model) View() tea.View {
    // If this string is identical to last frame...
    s := renderScreen(m)
    return tea.NewView(s)
}
```

**Result:** The cell-based renderer detects zero changes and sends **nothing** to the terminal. No CPU burn, no bandwidth waste.

### Practical Implication

Rendering a full 60×20 terminal at 30fps (33ms per frame) is **not a performance concern** in v2. The differential renderer handles it. You can safely:
- Re-render the full view every frame
- Animate colors, text, layout at 30fps
- Rely on the framework to optimize

**This is v2's major win over v1** — v1's line-based diff would send full lines on any pixel change; v2 sends only the cells that changed.

---

## 5. Easing Functions (Stdlib-Only)

Use these canonical formulas directly in your animation code. No external deps.

### EaseOutCubic (slow start, fast finish)

```go
func EaseOutCubic(t float64) float64 {
    // t ∈ [0.0, 1.0]
    t = t - 1
    return t*t*t + 1
}
```

**Use case:** Fade out, shrink, slide off-screen (decelerating exit).

### EaseInOutQuad (accelerate → decelerate)

```go
func EaseInOutQuad(t float64) float64 {
    // t ∈ [0.0, 1.0]
    if t < 0.5 {
        return 2 * t * t
    }
    t = t*2 - 1
    return -0.5 * (t*(t-2) - 1)
}
```

**Use case:** Smooth movement (snappy start, gentle finish).

### Usage Example

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case FrameTickMsg:
        progress := float64(m.frameIdx) / float64(m.frameDurationFrames)
        eased := EaseInOutQuad(progress)
        
        // Apply to position/scale/opacity
        m.x = m.xStart + (m.xEnd - m.xStart) * eased
        
        return m, tea.Tick(/* ... */)
}
```

---

## 6. Critically Damped Spring (Stdlib-Only)

For smooth motion that reaches equilibrium **without oscillating**, even if the target overshoots.

### Setup

```go
type Spring struct {
    pos    float64  // Current position
    vel    float64  // Current velocity
    target float64  // Equilibrium position
    omega  float64  // Angular frequency (~3.0 for snappy feel)
}
```

**Critical damping formula:** damping coefficient `c = 2 * omega` (ensures ζ = 1, no oscillation).

### Step Function (Symplectic Euler Integration)

```go
func (s *Spring) Step(dt float64) {
    // Precompute time-scaled coefficients
    expTerm     := math.Exp(-s.omega * dt)
    timeExp     := dt * expTerm
    timeExpFreq := timeExp * s.omega
    
    // Position update coefficients
    posPosCoef := timeExpFreq + expTerm
    posVelCoef := timeExp
    
    // Velocity update coefficients
    velPosCoef := -s.omega * timeExpFreq
    velVelCoef := -timeExpFreq + expTerm
    
    // Apply updates (relative to target)
    oldPos := s.pos - s.target
    s.pos = oldPos*posPosCoef + s.vel*posVelCoef + s.target
    s.vel = oldPos*velPosCoef + s.vel*velVelCoef
}
```

### Animation Loop Integration

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case FrameTickMsg:
        dt := 0.033  // 33ms frame time, in seconds
        m.spring.target = m.targetValue
        m.spring.Step(dt)
        
        // Use m.spring.pos for rendering
        m.displayValue = m.spring.pos
        
        return m, tea.Tick(33*time.Millisecond, /* ... */)
}
```

### Tuning

- **omega ≈ 3.0:** Snappy (reaches target in ~0.5–1s)
- **omega ≈ 1.0:** Floaty (reaches target in ~1–2s)
- Higher omega = faster response, risk of jerky motion if dt is too large
- Lower omega = smoother but sluggish

### Why Critical Damping?

- **Reaches equilibrium in minimum time** (no oscillation to settle)
- Ideal for camera pans, value fading, transitions
- Robust to target changes mid-animation (smooth interruption)

---

## 7. V1 → V2 Differences (Animation-Relevant)

| Aspect | v1 | v2 | Impact |
|--------|----|----|--------|
| **View() return type** | `string` | `tea.View` struct | Enables cell-diff rendering; safe to re-render every frame |
| **`tea.Tick` signature** | — | `tea.Tick(d, fn func(time.Time) Msg) Cmd` | No change; still single-fire; must re-arm in Update |
| **`tea.Every`** | — | `tea.Every(d, fn func(time.Time) Msg) Cmd` | v2 addition; syncs to system clock (not used for independent animations) |
| **Renderer** | Line-based diff | Cell-based Cursed Renderer | Orders of magnitude faster; full-screen 30fps redraws are cheap |
| **Lip Gloss colors** | `color.Color` interface | Same | No changes to color API between v1 and v2 |

**Key finding:** No breaking changes to animation APIs between v1 and v2. The win is in rendering efficiency.

---

## 8. Recommended Architecture for Typeburn

### Phase Structure

1. **Define a frame message:**
   ```go
   type FrameTickMsg struct{ t time.Time }
   ```

2. **Add animation state to screen models** (e.g., TypingModel, ResultModel):
   ```go
   type TypingModel struct {
       // ... existing fields ...
       animFrameIdx       int
       animDurationFrames int
       isAnimating        bool
       animColor          color.Color
       animStyle          lipgloss.Style
   }
   ```

3. **Start animation on transition:**
   ```go
   func (m TypingModel) startAnimation() (TypingModel, tea.Cmd) {
       m.isAnimating = true
       m.animFrameIdx = 0
       return m, tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
           return FrameTickMsg{t: t}
       })
   }
   ```

4. **Update on frame tick:**
   ```go
   case FrameTickMsg:
       if !m.isAnimating {
           return m, nil  // Guard: shouldn't receive tick if not animating
       }
       
       progress := float64(m.animFrameIdx) / float64(m.animDurationFrames)
       eased := EaseInOutQuad(progress)
       
       m.animColor = interpolateRGB(m.colorFrom, m.colorTo, eased)
       m.animStyle = lipgloss.NewStyle().Foreground(m.animColor)
       
       m.animFrameIdx++
       if m.animFrameIdx >= m.animDurationFrames {
           m.isAnimating = false
           return m, nil  // Stop loop
       }
       
       return m, tea.Tick(33*time.Millisecond, /* ... */)
   ```

5. **Render with animated style:**
   ```go
   func (m TypingModel) View() tea.View {
       var s strings.Builder
       if m.isAnimating {
           s.WriteString(m.animStyle.Render("Animated text"))
       }
       s.WriteString(/* ... rest of view ... */)
       return tea.NewView(s.String())
   }
   ```

### File Organization

Keep animation helpers in a new `internal/animation/` package:

```
internal/animation/
├── easing.go              # EaseOutCubic, EaseInOutQuad
├── spring.go              # Spring struct + Step()
├── interpolate.go         # interpolateRGB()
└── animation.go           # Shared types (FrameTickMsg, etc.)
```

Or add them directly to `internal/ui/timer.go` (already animation-focused).

---

## Implementation Checklist

- [ ] Define `FrameTickMsg` type in `internal/ui/messages.go` (alongside existing `StartTestMsg`, etc.)
- [ ] Add easing functions to `internal/animation/easing.go` (or `internal/ui/easing.go`)
- [ ] Add `interpolateRGB()` to `internal/animation/colors.go`
- [ ] Add `Spring` struct + `Step()` to `internal/animation/spring.go` (optional; use if spring easing desired)
- [ ] Update screen models (e.g., `ResultModel`) to hold animation state
- [ ] Add frame-tick handler to screen `Update()` methods
- [ ] Call animation start on screen transition (e.g., when showing result)
- [ ] Update `View()` to use animated styles
- [ ] Run `make test-race` to verify no race conditions
- [ ] Test at 30fps (33ms ticks) on a slow terminal (SSH) to confirm rendering efficiency

---

## Unresolved Questions

1. **Animation triggering:** Should animation start automatically when a screen appears, or on-demand (e.g., via a keystroke)? Depends on UX design.

2. **Frame cadence flexibility:** Should all animations use 33ms ticks, or would some benefit from 50ms (20fps) to save bandwidth on slow terminals? Recommend 33ms for typing-test UX (results reveal fast); measure in testing.

3. **Interpolation accuracy:** The `interpolateRGB()` function divides by 257 to normalize uint32 (0–65535) to uint8 (0–255). Confirm rounding is acceptable or use proper scaling (multiply by 255, divide by 65535).

4. **Spring stiffness mapping:** No guidance on omega values for typical Typeburn use cases (result reveal, transition timing). Recommend empirical tuning with omega ∈ [1.0, 5.0].

5. **Simultaneous animations:** If multiple screen elements animate in parallel, do they share one `FrameTickMsg` or define separate tick types? Current Typeburn app.Model routes to one active screen at a time (no parallelism), so single `FrameTickMsg` should suffice. Revisit if multi-screen animation is added.

---

## Sources

- [Bubble Tea v2 pkg.go.dev — tea.Tick, tea.Every](https://pkg.go.dev/charm.land/bubbletea/v2)
- [Bubble Tea v2 Upgrade Guide — View() changes](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- [Bubble Tea v2 Release — Cursed Renderer (PR #1200)](https://github.com/charmbracelet/bubbletea/pull/1200)
- [Lip Gloss v2 pkg.go.dev — Color API](https://pkg.go.dev/charm.land/lipgloss/v2)
- [Gotween — Easing implementations](https://github.com/paddycarey/gotween/blob/master/gotween.go)
- [RyanJuckett — Critically Damped Springs](https://www.ryanjuckett.com/damped-springs/)
- [Typeburn timer.go — Existing Tick pattern](https://github.com/bavanchun/Typeburn/blob/main/internal/ui/timer.go)
- [Typeburn theme.go — Color and Role system](https://github.com/bavanchun/Typeburn/blob/main/internal/theme/theme.go)

package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSimpleUIRemainsNoninteractive(t *testing.T) {
	t.Parallel()

	model := NewUnifiedUI(ModeSimple, "https://example.com", t.TempDir(), 2, 1)
	if model.IsBubbletea() {
		t.Fatal("simple UI must remain noninteractive")
	}
	if model.Init() != nil {
		t.Fatal("simple UI must not start terminal commands")
	}
}

func TestStandardUIV2Lifecycle(t *testing.T) {
	t.Parallel()

	model := newStandardUI("https://example.com", t.TempDir(), 2, 1)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = updated.(*StandardUI)
	if !model.ready || model.width != 100 || model.height != 30 {
		t.Fatalf("resize state = ready %t, %dx%d; want ready 100x30", model.ready, model.width, model.height)
	}

	updated, _ = model.Update(StatsMsg{PagesVisited: 3, PagesFailed: 1})
	model = updated.(*StandardUI)
	updated, _ = model.Update(DoneMsg{})
	model = updated.(*StandardUI)
	view := model.View()
	if !model.done || !view.AltScreen || view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("done = %t, terminal mode = alt %t, mouse %v", model.done, view.AltScreen, view.MouseMode)
	}

	_, quitCmd := model.Update(tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if quitCmd == nil {
		t.Fatal("q must return a quit command")
	}
}

func TestEnhancedUIV2InteractionLifecycle(t *testing.T) {
	t.Parallel()

	model := newEnhancedUI("https://example.com", t.TempDir(), 2, 2)
	if model.Init() == nil {
		t.Fatal("enhanced UI must initialize animation commands")
	}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = updated.(*EnhancedUI)
	if !model.ready || model.viewport.Width() != 120 {
		t.Fatalf("resize state = ready %t, viewport width %d", model.ready, model.viewport.Width())
	}
	view := model.View()
	if !view.AltScreen || view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("terminal mode = alt %t, mouse %v", view.AltScreen, view.MouseMode)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: '?', Text: "?"}))
	model = updated.(*EnhancedUI)
	if !model.showHelp {
		t.Fatal("? must open help")
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	model = updated.(*EnhancedUI)
	if model.showHelp {
		t.Fatal("escape must close help")
	}

	updated, _ = model.Update(DoneMsg{})
	model = updated.(*EnhancedUI)
	if !model.done {
		t.Fatal("done message must enter completion state")
	}
	_, quitCmd := model.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quitCmd == nil {
		t.Fatal("ctrl+c must return a quit command")
	}
}

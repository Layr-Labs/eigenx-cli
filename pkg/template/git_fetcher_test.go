package template_test

import (
	"context"
	"os/exec"
	"sync"
	"testing"

	"github.com/Layr-Labs/eigenx-cli/pkg/common/iface"
	"github.com/Layr-Labs/eigenx-cli/pkg/common/logger"
	"github.com/Layr-Labs/eigenx-cli/pkg/template"
)

// mockRunnerSuccess always returns a Cmd that exits 0
type mockRunnerSuccess struct{}

func (mockRunnerSuccess) CommandContext(_ context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.Command("true")
}

// mockRunnerFail always returns a Cmd that exits 1
type mockRunnerFail struct{}

func (mockRunnerFail) CommandContext(_ context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.Command("false")
}

// mockRunnerProgress emits "git clone"-style progress on stderr
type mockRunnerProgress struct{}

func (mockRunnerProgress) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	// This shell script writes two progress lines then exits 0
	script := `
      >&2 echo "Cloning into 'dest'..."
      >&2 echo "Receiving objects:  50%"
      sleep 0.01
      >&2 echo "Receiving objects: 100%"
      exit 0
    `
	return exec.CommandContext(ctx, "bash", "-c", script)
}

// spyTrackerDedup records only the latest Set() per module.
type spyTrackerDedup struct {
	mu    sync.Mutex
	order []string
	byID  map[string]struct {
		Pct   int
		Label string
	}
}

func newSpyTrackerDedup() *spyTrackerDedup {
	return &spyTrackerDedup{
		order: make([]string, 0),
		byID: make(map[string]struct {
			Pct   int
			Label string
		}),
	}
}

func (s *spyTrackerDedup) Set(id string, pct int, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, seen := s.byID[id]; !seen {
		s.order = append(s.order, id)
	}
	s.byID[id] = struct {
		Pct   int
		Label string
	}{pct, label}
}

func (s *spyTrackerDedup) Render() {}

func (s *spyTrackerDedup) Clear() {}

func (s *spyTrackerDedup) ProgressRows() []iface.ProgressRow { return make([]iface.ProgressRow, 0) }

// getFetcherWithRunner returns a GitFetcher and its underlying LogProgressTracker.
func getFetcherWithRunner(r template.Runner) (*template.GitFetcher, *spyTrackerDedup) {
	client := template.NewGitClientWithRunner(r)
	log := logger.NewNoopLogger()

	// Inject our spyTracker instead of the real one:
	spy := newSpyTrackerDedup()
	progressLogger := logger.NewProgressLogger(log, spy)

	return &template.GitFetcher{
		Client: client,
		Logger: *progressLogger,
		Config: template.GitFetcherConfig{Verbose: false},
	}, spy
}

func TestFetchSucceedsWithMockRunner(t *testing.T) {
	f, _ := getFetcherWithRunner(mockRunnerSuccess{})
	dir := t.TempDir()
	if err := f.Fetch(context.Background(), "any-url", "any-ref", dir); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestFetchFailsWhenCloneFails(t *testing.T) {
	f, _ := getFetcherWithRunner(mockRunnerFail{})
	dir := t.TempDir()
	err := f.Fetch(context.Background(), "any-url", "any-ref", dir)
	if err == nil {
		t.Fatal("expected error when git clone fails")
	}
}

func TestReporterGetsProgressEvents(t *testing.T) {
	// Build a client that emits 50% then 100% on stderr
	f, tracker := getFetcherWithRunner(mockRunnerProgress{})
	dir := t.TempDir()
	if err := f.Fetch(context.Background(), "irrelevant", "irrelevant", dir); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Inspect the tracker: it only logs on 100%
	rows := tracker.order

	// Expectations after successful run
	if len(rows) != 2 {
		t.Fatalf("expected 2 progress row, got %d", len(rows))
	}
	if tracker.byID[rows[0]].Pct != 100 {
		t.Errorf("expected the 100%% event, got %+v", rows[0])
	}
	if tracker.byID[rows[1]].Pct != 100 {
		t.Errorf("expected the 100%% event, got %+v", rows[0])
	}
}

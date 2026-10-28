package scheduler

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunEnforcesIndependentStageWorkerLimits(t *testing.T) {
	var encodeActive, subtitleActive atomic.Int32
	var encodeMax, subtitleMax atomic.Int32
	tasks := make([]Task, 0, 12)
	for index := 0; index < 6; index++ {
		tasks = append(tasks,
			Task{Source: "video", Stage: Compression, Run: trackedTask(&encodeActive, &encodeMax)},
			Task{Source: "video", Stage: Transcription, Run: trackedTask(&subtitleActive, &subtitleMax)},
		)
	}

	results, err := Run(context.Background(), tasks, Limits{EncodeWorkers: 2, SubtitleWorkers: 3}, nil)
	if err != nil {
		t.Fatalf("run scheduler: %v", err)
	}
	if len(results) != len(tasks) {
		t.Fatalf("got %d results, want %d", len(results), len(tasks))
	}
	if got := encodeMax.Load(); got != 2 {
		t.Errorf("maximum concurrent encodes = %d, want 2", got)
	}
	if got := subtitleMax.Load(); got != 3 {
		t.Errorf("maximum concurrent transcriptions = %d, want 3", got)
	}
}

func TestRunSchedulesStagesIndependentlyAndReturnsInputOrder(t *testing.T) {
	started := make(chan Stage, 2)
	release := make(chan struct{})
	task := func(source string, stage Stage) Task {
		return Task{Source: source, Stage: stage, Run: func(context.Context) error {
			started <- stage
			<-release
			return nil
		}}
	}
	tasks := []Task{
		task("clip-a", Compression),
		task("clip-a", Transcription),
	}
	resultCh := make(chan []Result, 1)
	errCh := make(chan error, 1)
	go func() {
		results, err := Run(context.Background(), tasks, Limits{EncodeWorkers: 1, SubtitleWorkers: 1}, nil)
		resultCh <- results
		errCh <- err
	}()

	seen := map[Stage]bool{}
	for range 2 {
		select {
		case stage := <-started:
			seen[stage] = true
		case <-time.After(time.Second):
			close(release)
			t.Fatal("compression and transcription did not both start before either was released")
		}
	}
	close(release)
	results := <-resultCh
	if err := <-errCh; err != nil {
		t.Fatalf("run scheduler: %v", err)
	}
	if !seen[Compression] || !seen[Transcription] {
		t.Fatalf("stages did not run concurrently: %#v", seen)
	}
	if got := []Stage{results[0].Stage, results[1].Stage}; !reflect.DeepEqual(got, []Stage{Compression, Transcription}) {
		t.Fatalf("results are not in input order: %v", got)
	}
}

func TestRunReportsQueuedRunningAndTerminalStatesAndKeepsStageFailuresIndependent(t *testing.T) {
	wantFailure := errors.New("encoder failed")
	tasks := []Task{
		{Source: "clip.mkv", Stage: Compression, Run: func(context.Context) error { return wantFailure }},
		{Source: "clip.mkv", Stage: Transcription, Run: func(context.Context) error { return nil }},
	}
	var mu sync.Mutex
	var events []Event
	results, err := Run(context.Background(), tasks, Limits{EncodeWorkers: 1, SubtitleWorkers: 1}, func(event Event) error {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("run scheduler: %v", err)
	}
	if results[0].State != Failed || !errors.Is(results[0].Err, wantFailure) {
		t.Fatalf("compression result = %+v, want failed encoder result", results[0])
	}
	if results[1].State != Completed || results[1].Err != nil {
		t.Fatalf("transcription result = %+v, want completed result", results[1])
	}
	for _, stage := range []Stage{Compression, Transcription} {
		states := statesFor(events, stage)
		if !reflect.DeepEqual(states, []State{Queued, Running, map[bool]State{true: Failed, false: Completed}[stage == Compression]}) {
			t.Errorf("%s states = %v", stage, states)
		}
	}
}

func TestRunRejectsInvalidWorkerLimits(t *testing.T) {
	_, err := Run(context.Background(), nil, Limits{EncodeWorkers: 0, SubtitleWorkers: 1}, nil)
	if err == nil {
		t.Fatal("expected zero encode workers to be rejected")
	}
}

func trackedTask(active, maximum *atomic.Int32) func(context.Context) error {
	return func(context.Context) error {
		current := active.Add(1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return nil
	}
}

func statesFor(events []Event, stage Stage) []State {
	var states []State
	for _, event := range events {
		if event.Stage == stage {
			states = append(states, event.State)
		}
	}
	return states
}

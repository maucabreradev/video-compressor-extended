// Package scheduler runs independent processing stages under separate limits.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Stage identifies one independently scheduled operation for a media source.
type Stage string

const (
	Compression   Stage = "compression"
	Transcription Stage = "transcription"
)

// State describes a task's lifecycle.
type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Completed State = "completed"
	Failed    State = "failed"
)

// Task is one stage of work for a source file.
type Task struct {
	Source string
	Stage  Stage
	Run    func(context.Context) error
}

// Limits independently bound active compression and transcription tasks.
type Limits struct {
	EncodeWorkers   int
	SubtitleWorkers int
}

// Event reports one transition in a task's lifecycle.
type Event struct {
	Index  int
	Source string
	Stage  Stage
	State  State
	Err    error
}

// Result contains the terminal state for a task. Results retain the same order
// as the input tasks regardless of the order in which workers finish.
type Result struct {
	Index  int
	Source string
	Stage  Stage
	State  State
	Err    error
}

// Run executes tasks with separate worker pools for compression and
// transcription. Task errors are recorded in results; callback errors are
// joined and returned after all tasks finish.
func Run(ctx context.Context, tasks []Task, limits Limits, onEvent func(Event) error) ([]Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("scheduler context must not be nil")
	}
	if limits.EncodeWorkers < 1 {
		return nil, fmt.Errorf("encode workers must be at least 1, got %d", limits.EncodeWorkers)
	}
	if limits.SubtitleWorkers < 1 {
		return nil, fmt.Errorf("subtitle workers must be at least 1, got %d", limits.SubtitleWorkers)
	}

	results := make([]Result, len(tasks))
	var compressionTasks, transcriptionTasks []int
	for index, task := range tasks {
		if task.Source == "" {
			return nil, fmt.Errorf("task %d source must not be empty", index)
		}
		if task.Run == nil {
			return nil, fmt.Errorf("task %d run function must not be nil", index)
		}
		results[index] = Result{Index: index, Source: task.Source, Stage: task.Stage, State: Queued}
		switch task.Stage {
		case Compression:
			compressionTasks = append(compressionTasks, index)
		case Transcription:
			transcriptionTasks = append(transcriptionTasks, index)
		default:
			return nil, fmt.Errorf("task %d has unsupported stage %q", index, task.Stage)
		}
	}

	var eventMu sync.Mutex
	var callbackErrors []error
	emit := func(event Event) {
		if onEvent == nil {
			return
		}
		eventMu.Lock()
		defer eventMu.Unlock()
		if err := onEvent(event); err != nil {
			callbackErrors = append(callbackErrors, fmt.Errorf("report %s state for %q: %w", event.State, event.Source, err))
		}
	}
	for index, task := range tasks {
		emit(Event{Index: index, Source: task.Source, Stage: task.Stage, State: Queued})
	}

	var workers sync.WaitGroup
	runPool := func(indices []int, limit int) {
		if len(indices) == 0 {
			return
		}
		workerCount := min(limit, len(indices))
		jobs := make(chan int, len(indices))
		for _, index := range indices {
			jobs <- index
		}
		close(jobs)
		for range workerCount {
			workers.Add(1)
			go func() {
				defer workers.Done()
				for index := range jobs {
					task := tasks[index]
					emit(Event{Index: index, Source: task.Source, Stage: task.Stage, State: Running})
					err := task.Run(ctx)
					state := Completed
					if err != nil {
						state = Failed
					}
					results[index] = Result{
						Index: index, Source: task.Source, Stage: task.Stage,
						State: state, Err: err,
					}
					emit(Event{Index: index, Source: task.Source, Stage: task.Stage, State: state, Err: err})
				}
			}()
		}
	}

	runPool(compressionTasks, limits.EncodeWorkers)
	runPool(transcriptionTasks, limits.SubtitleWorkers)
	workers.Wait()

	eventMu.Lock()
	defer eventMu.Unlock()
	return results, errors.Join(callbackErrors...)
}

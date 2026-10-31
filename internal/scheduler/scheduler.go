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

// Stage values identify the independent compression and subtitle work for a source.
const (
	Compression   Stage = "compression"
	Transcription Stage = "transcription"
)

// State describes a task's lifecycle.
type State string

// State values describe queued dispatch, execution, and terminal task outcomes.
// Skipped means the existing final artifact passed validation and was reused.
const (
	Queued    State = "queued"
	Running   State = "running"
	Completed State = "completed"
	Failed    State = "failed"
	Cancelled State = "cancelled"
	Skipped   State = "skipped"
)

// Task is one stage of work for a source file.
type Task struct {
	Source      string
	Destination string
	Stage       Stage
	Run         func(context.Context) error
	Skip        bool
}

// Limits independently bound active compression and transcription tasks.
type Limits struct {
	EncodeWorkers   int
	SubtitleWorkers int
}

// Event reports one transition in a task's lifecycle.
type Event struct {
	Index       int
	Source      string
	Destination string
	Stage       Stage
	State       State
	Err         error
}

// Result contains the terminal state for a task. Results retain the same order
// as the input tasks regardless of the order in which workers finish.
type Result struct {
	Index       int
	Source      string
	Destination string
	Stage       Stage
	State       State
	Err         error
}

// Run executes tasks with separate worker pools for compression and
// transcription. Task errors are recorded in results; callback errors are
// joined and returned after all tasks finish.
func Run(ctx context.Context, tasks []Task, limits Limits, onEvent func(Event) error) ([]Result, error) {
	return RunWithControl(ctx, tasks, limits, nil, onEvent)
}

// RunWithControl executes tasks under separate stage limits while honoring
// pauses before each task is dispatched.
func RunWithControl(
	ctx context.Context,
	tasks []Task,
	limits Limits,
	control *DispatchControl,
	onEvent func(Event) error,
) ([]Result, error) {
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
		if task.Run == nil && !task.Skip {
			return nil, fmt.Errorf("task %d run function must not be nil", index)
		}
		results[index] = Result{
			Index: index, Source: task.Source, Destination: task.Destination,
			Stage: task.Stage, State: Queued,
		}
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
		emit(Event{
			Index: index, Source: task.Source, Destination: task.Destination,
			Stage: task.Stage, State: Queued,
		})
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
				for {
					if ctx.Err() != nil {
						return
					}
					var index int
					var ok bool
					select {
					case <-ctx.Done():
						return
					case index, ok = <-jobs:
						if !ok {
							return
						}
					}
					if ctx.Err() != nil {
						continue
					}
					if err := control.wait(ctx); err != nil {
						continue
					}
					if ctx.Err() != nil {
						continue
					}
					task := tasks[index]
					if task.Skip {
						results[index] = Result{
							Index: index, Source: task.Source, Destination: task.Destination,
							Stage: task.Stage, State: Skipped,
						}
						emit(Event{
							Index: index, Source: task.Source, Destination: task.Destination,
							Stage: task.Stage, State: Skipped,
						})
						continue
					}
					emit(Event{
						Index: index, Source: task.Source, Destination: task.Destination,
						Stage: task.Stage, State: Running,
					})
					err := task.Run(ctx)
					state := Completed
					if err != nil {
						state = Failed
						if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							state = Cancelled
						}
					}
					results[index] = Result{
						Index: index, Source: task.Source, Destination: task.Destination, Stage: task.Stage,
						State: state, Err: err,
					}
					emit(Event{
						Index: index, Source: task.Source, Destination: task.Destination,
						Stage: task.Stage, State: state, Err: err,
					})
				}
			}()
		}
	}

	runPool(compressionTasks, limits.EncodeWorkers)
	runPool(transcriptionTasks, limits.SubtitleWorkers)
	workers.Wait()
	for index, result := range results {
		if result.State != Queued {
			continue
		}
		result.State = Cancelled
		result.Err = ctx.Err()
		if result.Err == nil {
			result.Err = context.Canceled
		}
		results[index] = result
		emit(Event{
			Index: index, Source: result.Source, Destination: result.Destination,
			Stage: result.Stage, State: Cancelled, Err: result.Err,
		})
	}

	eventMu.Lock()
	defer eventMu.Unlock()
	return results, errors.Join(callbackErrors...)
}

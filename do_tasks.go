package flowmatic

import (
	"github.com/carlmjohnson/deque"
)

// Manager is a function that serially examines Task results to see if it produced any new Inputs.
// Returning false will halt the processing of future tasks.
type Manager[Input, Output any] func(Input, Output, error) (tasks []Input, ok bool)

// Task is a function that can concurrently transform an input into an output.
type Task[Input, Output any] func(in Input) (out Output, err error)

// DoTasks does tasks using n concurrent workers (or GOMAXPROCS workers if n < 1)
// which produce output consumed by a serially run manager.
// The manager should return a slice of new task inputs based on prior task results,
// or return false to halt processing.
// If a task panics during execution,
// the panic will be caught and rethrown in the parent Goroutine.
func DoTasks[Input, Output any](n int, task Task[Input, Output], manager Manager[Input, Output], initial ...Input) {
	in, out := start(n, task)
	defer func() {
		close(in)
		// drain any waiting tasks
		for range out {
		}
	}()
	queue := deque.Of(initial...)
	inflight := 0
	for inflight > 0 || queue.Len() > 0 {
		inch := in
		item, ok := queue.Head()
		if !ok {
			inch = nil
		}
		select {
		case inch <- item:
			inflight++
			queue.PopHead()
		case r := <-out:
			inflight--
			if r.Panic != nil {
				panic(r.Panic)
			}
			items, ok := manager(r.In, r.Out, r.Err)
			if !ok {
				return
			}
			queue.Append(items...)
		}
	}
}

// DoTasksLIFO is the same as DoTasks except tasks in the task queue are
// evaluated in last in, first out order.
func DoTasksLIFO[Input, Output any](n int, task Task[Input, Output], manager Manager[Input, Output], initial ...Input) {
	in, out := start(n, task)
	defer func() {
		close(in)
		// drain any waiting tasks
		for range out {
		}
	}()
	queue := deque.Of(initial...)
	inflight := 0
	for inflight > 0 || queue.Len() > 0 {
		inch := in
		item, ok := queue.Tail()
		if !ok {
			inch = nil
		}
		select {
		case inch <- item:
			inflight++
			queue.PopTail()
		case r := <-out:
			inflight--
			if r.Panic != nil {
				panic(r.Panic)
			}
			items, ok := manager(r.In, r.Out, r.Err)
			if !ok {
				return
			}
			queue.Append(items...)
		}
	}
}
/*
Package once contains the implementation details for the once strategy.

The once strategy executes the batch function for every call to Thunk or ThunkMany.
It can be configured to call the batch function when Thunk or ThunkMany is called, or
the batch function can be called in a background go routine. Defaults to executing
per call to Thunk/ThunkMany.
*/
package once

import (
	"context"

	"github.com/andy9775/dataloader"
)

// Options contains the strategy configuration
type Options struct {
	// InBackground specifies if the batch function should be executed in a background thread.
	InBackground bool // default to false
}

// NewOnceStrategy returns a new instance of the once strategy.
// The Once strategy calls the batch function for each call to the Thunk if InBackground is false.
// Otherwise it runs the batch function in a background go routine and blocks calls to Thunk or
// ThunkMany if the result is not yet fetched.
func NewOnceStrategy(batch dataloader.BatchFunction, opts Options) func(int) dataloader.Strategy {
	return func(_ int) dataloader.Strategy {
		return &onceStrategy{
			batchFunc: batch,
			options:   opts,
		}
	}
}

type onceStrategy struct {
	batchFunc dataloader.BatchFunction

	options Options
}

// Load returns a Thunk which either calls the batch function when invoked or waits for a result from a
// background go routine (blocking if no data is available).
func (s *onceStrategy) Load(ctx context.Context, key dataloader.Key) dataloader.Thunk {
	if s.options.InBackground {
		resultChan := make(chan dataloader.Result)

		go func() {
			resultChan <- (*s.batchFunc(ctx, dataloader.NewKeysWith(key))).GetValue(key)
		}()

		// call batch in background and block util it returns
		return func() dataloader.Result {
			return <-resultChan
		}
	}

	// call batch when thunk is called
	return func() dataloader.Result {
		return (*s.batchFunc(ctx, dataloader.NewKeysWith(key))).GetValue(key)
	}

}

// LoadMany returns a ThunkMany which either calls the batch function when invoked or waits for a result from
// a background go routine (blocking if no data is available).
func (s *onceStrategy) LoadMany(ctx context.Context, keyArr ...dataloader.Key) dataloader.ThunkMany {
	if s.options.InBackground {
		resultChan := make(chan dataloader.ResultMap)

		go func() {
			resultChan <- *s.batchFunc(ctx, dataloader.NewKeysWith(keyArr...))
		}()

		// call batch in background and block util it returnsS
		return func() dataloader.ResultMap {
			return <-resultChan
		}
	}

	// call batch when thunk is called
	return func() dataloader.ResultMap {
		return *s.batchFunc(ctx, dataloader.NewKeysWith(keyArr...))
	}

}

// LoadNoOp has no internal implementation since the once strategy doesn't track the number of calls to
// Load or Loadmany
func (*onceStrategy) LoadNoOp(context.Context) {}

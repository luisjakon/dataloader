package standard_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/andy9775/dataloader"
	"github.com/andy9775/dataloader/strategies/standard"
	"github.com/bouk/monkey"
	"github.com/stretchr/testify/assert"
)

// ============================================== test constants =============================================
const TEST_TIMEOUT time.Duration = time.Millisecond * 500

// ==================================== implement concrete keys interface ====================================
type PrimaryKey int

func (p PrimaryKey) String() string {
	return strconv.Itoa(int(p))
}

func (p PrimaryKey) Raw() interface{} {
	return p
}

// =============================================== test helpers ==============================================

// getBatchFunction returns a generic batch function which returns the provided result and calls the provided
// callback function
func getBatchFunction(cb func(dataloader.Keys), result string) dataloader.BatchFunction {
	return func(ctx context.Context, keys dataloader.Keys) *dataloader.ResultMap {
		cb(keys)
		m := dataloader.NewResultMap(1)
		for _, k := range keys.Keys() {
			key := k.(PrimaryKey)
			m.Set(
				key,
				dataloader.Result{
					// ensure each result in ResultMap is uniquely identifiable
					Result: fmt.Sprintf("%s_%s", key, result),
					Err:    nil,
				},
			)
		}
		return &m
	}
}

// timeout will panic if a test takes more than a defined time.
// `timeoutChannel chan struct{}` should be closed when the test completes in order to
// signal that it completed within the defined time
func timeout(t *testing.T, timeoutChannel chan struct{}, after time.Duration) {
	go func() {
		time.Sleep(after)
		select {
		case <-timeoutChannel:
			return
		default:
			panic(fmt.Sprintf("%s took too long to execute", t.Name()))
		}
	}()
}

type mockLogger struct {
	logMsgs []string
	m       sync.Mutex
}

func (l *mockLogger) Log(v ...interface{}) {
	l.m.Lock()
	defer l.m.Unlock()

	for _, value := range v {
		switch i := value.(type) {
		case string:
			l.logMsgs = append(l.logMsgs, i)
		default:
			panic("mock logger only takes single log string")
		}
	}
}

func (l *mockLogger) Logf(format string, v ...interface{}) {
	l.m.Lock()
	defer l.m.Unlock()

	l.logMsgs = append(l.logMsgs, fmt.Sprintf(format, v...))
}

func (l *mockLogger) Messages() []string {
	l.m.Lock()
	defer l.m.Unlock()

	result := make([]string, len(l.logMsgs))
	copy(result, l.logMsgs)
	return result
}

// ================================================== tests ==================================================

// ================================================ no timeout ===============================================

// TestLoadNoTimeout tests calling the load function without timing out
func TestLoadNoTimeout(t *testing.T) {
	// setup
	wg := sync.WaitGroup{} // ensure batch function called before asserting
	wg.Add(1)

	// blockWG blocks the callback function allowing the test to assert that Load() function doesn't block
	blockWG := sync.WaitGroup{}
	blockWG.Add(1)
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT)

	var k []interface{}
	callCount := 0
	expectedResult := "batch_on_load"
	cb := func(keys dataloader.Keys) {
		blockWG.Wait()
		callCount += 1
		k = keys.RawKeys()
		close(closeChan)
		wg.Done()
	}

	timedOut := false
	monkey.Patch(time.After, func(t time.Duration) <-chan time.Time {
		defer monkey.Unpatch(time.After)
		toChan := make(chan time.Time, 1)

		go func() {
			time.Sleep(t)
			timedOut = true
			toChan <- time.Now()
		}()

		return toChan
	})

	key := PrimaryKey(1)
	key2 := PrimaryKey(2)

	/*
		ensure the loader doesn't call batch after timeout. If it does, the test will timeout and panic
	*/
	timeout := TEST_TIMEOUT * 5
	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy(standard.WithTimeout(timeout))(3, batch) // expects 3 load calls

	// invoke/assert
	strategy.Load(context.Background(), key)           // --------- Load 		 - call 1
	thunk := strategy.Load(context.Background(), key2) // --------- Load 		 - call 2
	strategy.LoadNoOp(context.Background())            // --------- LoadNoOp - call 3
	assert.Equal(t, 0, callCount, "Load() not expected to block or call batch function")
	blockWG.Done()
	wg.Wait()

	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
	assert.Equal(t, 2, len(k), "Expected to be called with 2 keys")

	r, ok := thunk()
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		r.Result.(string),
		"Expected result from thunk()",
	)

	assert.False(t, timedOut, "Expected loader not to timeout")

	// test double call to thunk
	r, ok = thunk()
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		r.Result.(string),
		"Expected result from thunk()",
	)
	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
}

// TestLoadManyNoTimeout tests calling the load function without timing out
func TestLoadManyNoTimeout(t *testing.T) {
	// setup
	wg := sync.WaitGroup{} // ensure batch function called before asserting
	wg.Add(1)

	// blockWG blocks the callback function allowing the test to assert that Load() function doesn't block
	blockWG := sync.WaitGroup{}
	blockWG.Add(1)
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT)

	var k []interface{}
	callCount := 0
	expectedResult := "batch_on_load_many"
	cb := func(keys dataloader.Keys) {
		blockWG.Wait()
		callCount += 1
		k = keys.RawKeys()
		close(closeChan)
		wg.Done()
	}

	timedOut := false
	monkey.Patch(time.After, func(t time.Duration) <-chan time.Time {
		defer monkey.Unpatch(time.After)
		toChan := make(chan time.Time, 1)

		go func() {
			time.Sleep(t)
			timedOut = true
			toChan <- time.Now()
		}()

		return toChan
	})

	key := PrimaryKey(1)
	key2 := PrimaryKey(2)
	key3 := PrimaryKey(3)

	/*
		ensure the loader doesn't call batch after timeout. If it does, the test will timeout and panic
	*/
	timeout := TEST_TIMEOUT * 5
	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy(standard.WithTimeout(timeout))(3, batch) // expects 3 load calls

	// invoke/assert
	strategy.LoadMany(context.Background(), key)                 // --------- LoadMany 		 - call 1
	thunk := strategy.LoadMany(context.Background(), key2, key3) // --------- LoadMany 		 - call 2
	strategy.LoadNoOp(context.Background())                      // --------- LoadNoOp 		 - call 3
	assert.Equal(t, 0, callCount, "Load() not expected to block or call batch function")
	blockWG.Done()
	wg.Wait()

	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
	assert.Equal(t, 3, len(k), "Expected to be called with 2 keys")

	r := thunk()
	returned, ok := r.GetValue(key2)
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		returned.Result.(string),
		"Expected result from thunk()",
	)

	assert.False(t, timedOut, "Expected loader not to timeout")

	// test double call to thunk
	r = thunk()
	returned, ok = r.GetValue(key2)
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		returned.Result.(string),
		"Expected result from thunk()",
	)
	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
}

// ================================================= timeout =================================================

// TestLoadTimeout tests that the first load call is performed after a timeout and the second occurs when the
// thunk is called.
func TestLoadTimeout(t *testing.T) {
	// setup
	wg := sync.WaitGroup{} // ensure batch function called before asserting
	wg.Add(1)

	// blockWG blocks the callback function allowing the test to assert that Load() function doesn't block
	blockWG := sync.WaitGroup{}
	blockWG.Add(1)
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT)

	var k []interface{}
	callCount := 0
	expectedResult := "batch_on_timeout_load"
	cb := func(keys dataloader.Keys) {
		blockWG.Wait()
		callCount += 1
		k = keys.RawKeys()
		if callCount == 2 {
			close(closeChan)
		}
		wg.Done()
	}

	toWG := sync.WaitGroup{}
	toWG.Add(1)
	timedOut := false
	monkey.Patch(time.After, func(t time.Duration) <-chan time.Time {
		defer monkey.Unpatch(time.After)
		toChan := make(chan time.Time, 1)

		go func() {
			time.Sleep(t)
			timedOut = true
			toWG.Done()
			toChan <- time.Now()
		}()

		return toChan
	})

	key := PrimaryKey(1)
	key2 := PrimaryKey(2)

	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy()(3, batch) // expects 3 load calls

	// invoke/assert

	thunk2 := strategy.Load(context.Background(), key2) // --------- Load 		 - call 1
	strategy.LoadNoOp(context.Background())             // --------- LoadNoOp  - call 2
	assert.Equal(t, 0, callCount, "Load() not expected to block or call batch function")
	blockWG.Done()
	wg.Wait()

	toWG.Wait()
	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
	assert.Equal(t, 1, len(k), "Expected to be called with 1 key")
	assert.True(t, timedOut, "Expected loader to timeout")

	r, ok := thunk2()
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		r.Result.(string),
		"Expected result from thunk()",
	)

	// don't wait below - callback is not executing in background go routine - ensure wg doesn't go negative
	wg.Add(1)
	thunk := strategy.Load(context.Background(), key) // --------- Load 		 - call 3
	r, ok = thunk()
	assert.True(t, ok, "Expected result to have been found")

	// called once in go routine after timeout, once in thunk
	assert.Equal(t, 2, callCount, "Batch function expected to be called twice")
	assert.Equal(t,
		fmt.Sprintf("1_%s", expectedResult),
		r.Result.(string),
		"Expected result from thunk",
	)

	// test double call to thunk
	r, ok = thunk()
	assert.True(t, ok, "Expected result to have been found")

	// called once in go routine after timeout, once in thunk
	assert.Equal(t, 2, callCount, "Batch function expected to be called twice")
	assert.Equal(t,
		fmt.Sprintf("1_%s", expectedResult),
		r.Result.(string),
		"Expected result from thunk",
	)
}

// TestLoadManyTimeout tests that the first load call is performed after a timeout and the second
// occurs when the thunk is called.
func TestLoadManyTimeout(t *testing.T) {
	// setup
	wg := sync.WaitGroup{} // ensure batch function called before asserting
	wg.Add(1)

	// blockWG blocks the callback function allowing the test to assert that Load() function doesn't block
	blockWG := sync.WaitGroup{}
	blockWG.Add(1)
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT)

	var k []interface{}
	callCount := 0
	expectedResult := "batch_on_timeout_load_many"
	cb := func(keys dataloader.Keys) {
		blockWG.Wait()
		callCount += 1
		k = keys.RawKeys()
		if callCount == 2 {
			close(closeChan)
		}
		wg.Done()
	}

	toWG := sync.WaitGroup{}
	toWG.Add(1)
	timedOut := false
	monkey.Patch(time.After, func(t time.Duration) <-chan time.Time {
		defer monkey.Unpatch(time.After)
		toChan := make(chan time.Time, 1)

		go func() {
			time.Sleep(t)
			timedOut = true
			toWG.Done()
			toChan <- time.Now()
		}()

		return toChan
	})

	key := PrimaryKey(1)
	key2 := PrimaryKey(2)
	key3 := PrimaryKey(3)

	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy()(3, batch) // expects 3 load calls

	// invoke/assert

	thunkMany2 := strategy.LoadMany(context.Background(), key2) // --------- LoadMany 		 - call 1
	strategy.LoadNoOp(context.Background())                     // --------- LoadNoOp      - call 2
	assert.Equal(t, 0, callCount, "Load() not expected to block or call batch function")
	blockWG.Done()
	wg.Wait()

	toWG.Wait()
	assert.Equal(t, 1, callCount, "Batch function expected to be called once")
	assert.Equal(t, 1, len(k), "Expected to be called with 1 key")
	assert.True(t, timedOut, "Expected loader to timeout")

	r := thunkMany2()
	returned, ok := r.GetValue(key2)
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(
		t,
		fmt.Sprintf("2_%s", expectedResult),
		returned.Result.(string),
		"Expected result from thunkMany()",
	)

	// don't wait below - callback is not executing in background go routine - ensure wg doesn't go negative
	wg.Add(1)
	thunkMany := strategy.LoadMany(context.Background(), key, key3) // --------- LoadMany 		 - call 3
	r = thunkMany()

	// called once in go routine after timeout, once in thunkMany
	assert.Equal(t, 2, callCount, "Batch function expected to be called twice")
	returned, ok = r.GetValue(key3)
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(t,
		fmt.Sprintf("3_%s", expectedResult),
		returned.Result.(string),
		"Expected result from thunkMany",
	)
	assert.Equal(t, 2, len(k), "Expected to be called with 2 keys") // second function call

	// test double call to thunk
	r = thunkMany()

	// called once in go routine after timeout, once in thunkMany
	assert.Equal(t, 2, callCount, "Batch function expected to be called twice")
	returned, ok = r.GetValue(key3)
	assert.True(t, ok, "Expected result to have been found")
	assert.Equal(t,
		fmt.Sprintf("3_%s", expectedResult),
		returned.Result.(string),
		"Expected result from thunkMany",
	)
}

// =========================================== cancellable context ===========================================

// TestCancellableContextLoad ensures that a call to cancel the context kills the background worker
func TestCancellableContextLoad(t *testing.T) {
	// setup
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT*3)

	callCount := 0
	expectedResult := "cancel_via_context"
	cb := func(keys dataloader.Keys) {
		callCount += 1
		close(closeChan)
	}

	key := PrimaryKey(1)
	log := mockLogger{logMsgs: make([]string, 2), m: sync.Mutex{}}
	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy(
		standard.WithLogger(&log),
	)(2, batch) // expected 2 load calls
	ctx, cancel := context.WithCancel(context.Background())

	// invoke
	cancel()
	thunk := strategy.Load(ctx, key)
	thunk()
	time.Sleep(100 * time.Millisecond)

	// assert
	assert.Equal(t, 0, callCount, "Batch should not have been called")
	m := log.Messages()
	assert.Equal(t, "worker cancelled", m[len(m)-1], "Expected worker to cancel and log exit")
}

// TestCancellableContextLoadMany ensures that a call to cancel the context kills the background worker
func TestCancellableContextLoadMany(t *testing.T) {
	// setup
	closeChan := make(chan struct{})
	timeout(t, closeChan, TEST_TIMEOUT*3)

	callCount := 0
	expectedResult := "cancel_via_context"
	cb := func(keys dataloader.Keys) {
		callCount += 1
		close(closeChan)
	}

	key := PrimaryKey(1)
	log := mockLogger{logMsgs: make([]string, 2), m: sync.Mutex{}}
	batch := getBatchFunction(cb, expectedResult)
	strategy := standard.NewStandardStrategy(
		standard.WithLogger(&log),
	)(2, batch) // expected 2 load calls
	ctx, cancel := context.WithCancel(context.Background())

	// invoke
	cancel()
	thunk := strategy.LoadMany(ctx, key)
	thunk()
	time.Sleep(100 * time.Millisecond)

	// assert
	assert.Equal(t, 0, callCount, "Batch should not have been called")
	m := log.Messages()
	assert.Equal(t, "worker cancelled", m[len(m)-1], "Expected worker to cancel and log exit")
}

// =============================================== result keys ===============================================
// TestKeyHandling ensure that the strategy properly handles unprocessed and nil keys
func TestKeyHandling(t *testing.T) {
	// setup
	expectedResult := map[PrimaryKey]interface{}{
		PrimaryKey(1): "valid_result",
		PrimaryKey(2): nil,
		PrimaryKey(3): "__skip__", // this key should be skipped by the batch function
	}

	batch := func(ctx context.Context, keys dataloader.Keys) *dataloader.ResultMap {
		m := dataloader.NewResultMap(2)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Keys()[i].(PrimaryKey)
			if expectedResult[key] != "__skip__" {
				m.Set(key, dataloader.Result{Result: expectedResult[key], Err: nil})
			}
		}
		return &m
	}

	// invoke/assert
	strategy := standard.NewStandardStrategy()(3, batch)

	// Load
	for key, expected := range expectedResult {
		thunk := strategy.Load(context.Background(), key)
		r, ok := thunk()

		switch expected.(type) {
		case string:
			if expected == "__skip__" {
				assert.False(t, ok, "Expected skipped result to not be found")
				assert.Nil(t, r.Result, "Expected skipped result to be nil")
			} else {
				assert.True(t, ok, "Expected processed result to be found")
				assert.Equal(t, r.Result, expected, "Expected result")
			}
		case nil:
			assert.True(t, ok, "Expected processed result to be found")
			assert.Nil(t, r.Result, "Expected result to be nil")
		}
	}

	// LoadMany
	thunkMany := strategy.LoadMany(context.Background(), PrimaryKey(1), PrimaryKey(2), PrimaryKey(3))
	for key, expected := range expectedResult {
		result := thunkMany()
		r, ok := result.GetValue(key)

		switch expected.(type) {
		case string:
			if expected == "__skip__" {
				assert.False(t, ok, "Expected skipped result to not be found")
				assert.Nil(t, r.Result, "Expected skipped result to be nil")
			} else {
				assert.True(t, ok, "Expected processed result to be found")
				assert.Equal(t, r.Result, expected, "Expected result")
			}
		case nil:
			assert.True(t, ok, "Expected processed result to be found")
			assert.Nil(t, r.Result, "Expected result to be nil")
		}

	}
}

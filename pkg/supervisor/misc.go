package supervisor

import (
	"fmt"
	"reflect"
	"time"
)

func nextDelay(delay time.Duration) time.Duration {
	switch {
	case delay <= 0:
		return 100 * time.Millisecond
	case delay < 3*time.Second:
		return delay * 2
	default:
		return 3 * time.Second
	}
}

// WorkFunc creates a work function from a function whose signature
// includes a process plus additional arguments.
func WorkFunc(fn interface{}, args ...interface{}) func(*Process) error {
	fnv := reflect.ValueOf(fn)
	return func(p *Process) error {
		vargs := []reflect.Value{reflect.ValueOf(p)}
		for _, a := range args {
			vargs = append(vargs, reflect.ValueOf(a))
		}
		result := fnv.Call(vargs)
		if len(result) != 1 {
			panic(fmt.Sprintf("unexpected result: %v", result))
		}
		v := result[0].Interface()
		if v != nil {
			err, ok := v.(error)
			if !ok {
				panic(fmt.Sprintf("unrecognized result type: %v", v))
			}
			return err
		}
		return nil
	}
}

package utils

import (
	"context"
	"log"
)

const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

// ReadN reads n values from a channel or is stopped
func ReadN[T any](stop <-chan struct{}, ch <-chan T, n int) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for count := 0; count < n; count++ {
			select {
			case <-stop:
				return
			case <-ch:
			}
		}
	}()
	return done
}

func TakeN[T any](stop <-chan struct{}, ch <-chan T, n int) <-chan T {
	taken := make(chan T)

	go func() {
		defer close(taken)
		for count := 0; count < n; count++ {
			select {
			case <-stop:
				return
			case val, open := <-ch:
				if !open {
					return
				}
				taken <- val
			}
		}
	}()

	return taken
}

func CtxOrDone(ctx context.Context, done <-chan struct{}) <-chan struct{} {
	terminated := make(chan struct{})
	go func() {
		defer close(terminated)
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		}
	}()
	return terminated
}

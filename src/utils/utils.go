package utils

import (
	"log"
)

const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

// Take reads n values from a channel or is stopped
func Take[T interface{}](stop <-chan struct{}, ch <-chan T, n int) <-chan struct{} {
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

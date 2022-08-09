package utils

import (
	"container/list"
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

func AllClosed[T interface{}](stop <-chan struct{}, chs []<-chan T) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		list := list.New()

		for ch := range chs {
			list.PushFront(ch)
		}

		for e := list.Front(); e != nil; e.Next() {
			select {
			case <-stop:
				return
			default:
			}
			if _, open := <-e.Value.(<-chan T); !open {
				list.Remove(e)
			}
		}
	}()
	return done
}

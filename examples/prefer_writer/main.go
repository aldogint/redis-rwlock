package main

import (
	"fmt"
	"sync"
	"time"

	rwlock "github.com/aldogint/redis-rwlock"
	"github.com/gomodule/redigo/redis"
)

const (
	readersCount    = 10
	writeIterations = 5
	writeDuration   = 500 * time.Millisecond
	writeInterval   = 2 * time.Second
)

type example struct {
	locker rwlock.Locker
	wg     sync.WaitGroup
	doneC  chan struct{}
}

func (e *example) WriteSharedData(sharedData *int) {
	e.wg.Add(1)
	go func() {
		for i := 0; i < writeIterations; i++ {
			err := e.locker.Write(func() {
				fmt.Printf("Writing...\n")
				time.Sleep(writeDuration)
				(*sharedData)++
				fmt.Printf("Write: %d\n", *sharedData)
			})
			if err != nil {
				fmt.Printf("Writing error: %v\n", err)
			}
			time.Sleep(writeInterval)
		}
		close(e.doneC)
		e.wg.Done()
	}()
}

func (e *example) ReadSharedData(sharedData *int) {
	e.wg.Add(1)
	go func() {
		for {
			select {
			case <-e.doneC:
				e.wg.Done()
				return
			default:
				err := e.locker.Read(func() {
					fmt.Printf("Read: %d\n", *sharedData)
				})
				if err != nil {
					fmt.Printf("Read error: %v\n", err)
				}
			}
		}
	}()
}

func (e *example) Wait() {
	e.wg.Wait()
}

func main() {
	var (
		sharedData = 0
		redisPool  = &redis.Pool{
			Dial: func() (redis.Conn, error) {
				rc, err := redis.Dial("tcp", ":6379")
				if err != nil {
					return nil, err
				}
				return rc, nil
			},
		}
		example = example{
			locker: rwlock.New(redisPool, "GLOBAL_LOCK", "READER_COUNT", "WRITER_INTENT", &rwlock.Options{}),
			doneC:  make(chan struct{}),
		}
	)
	for i := 0; i < readersCount; i++ {
		example.ReadSharedData(&sharedData)
	}
	example.WriteSharedData(&sharedData)
	example.Wait()
}

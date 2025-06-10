package main

import (
	"fmt"
	"time"
	"vk-internship/internal/pool"
)

func main() {
	p := pool.New(10)
	id1 := p.AddWorker()
	id2 := p.AddWorker()
	fmt.Println("Workers:", id1, id2)

	go func() {
		for i := 1; i <= 5; i++ {
			p.Submit(fmt.Sprintf("task-%d", i))
		}
	}()

	time.Sleep(500 * time.Millisecond)
	p.RemoveWorker(id1)
	fmt.Println("Removed worker", id1)

	p.AddWorker()
	p.Submit("last-task")

	time.Sleep(1 * time.Second)
	p.Shutdown()
}

package main

import (
	"fmt"
	"sync"

	"github.com/prgra/oping"
)

func main() {
	p, err := oping.New(oping.Conf{Workers: 10000})
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for y := 0; y < 255; y++ {
		for x := 0; x < 255; x++ {
			go func(x int, y int, wg *sync.WaitGroup, p *oping.Pinger) {
				wg.Add(1)
				st, err := p.Ping(fmt.Sprintf("10.128.%d.%d", y, x), 5)
				if err != nil {
					fmt.Println(err)
				}
				succes := 0
				for _, s := range st {
					if s.Recv {
						succes++
					}
				}
				if succes > 0 {
					fmt.Printf("10.128.%d.%d - %d\n", y, x, succes)
				}
				wg.Done()

			}(x, y, &wg, p)
		}
	}
	wg.Wait()
	p.Close()

}

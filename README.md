# oping

[![GoDoc](https://godoc.org/github.com/prgra/oping?status.svg)](https://godoc.org/github.com/prgra/oping)
[![Go Report Card](https://goreportcard.com/badge/github.com/prgra/oping)](https://goreportcard.com/report/github.com/prgra/go-oping)

A simple ICMP checker of alive hosts, based on [net-icmp](https://golang.org/x/net/icmp)

Main feature - one listener for all incomming packets.

example: 

```go
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

```

10000 workets, ping 5 packets
launch on MacBook Pro mid 2012
```shell
# go build && time sudo ./_example > alive
sudo ./_example > alive  24.90s user 23.21s system 48% cpu 1:39.61 total
```

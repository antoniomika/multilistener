# multilistener [![Go Reference](https://pkg.go.dev/badge/github.com/antoniomika/multilistener.svg)](https://pkg.go.dev/github.com/antoniomika/multilistener) [![Coverage Status](https://coveralls.io/repos/github/antoniomika/multilistener/badge.svg?branch=main)](https://coveralls.io/github/antoniomika/multilistener?branch=main)

A package to listen on multiple ports as a single listener.

## [How to use](https://play.golang.com/p/kIM3rw04UeS)

```golang
package main

import (
    "log"

    "github.com/antoniomika/multilistener"
)

func main() {
    m, err := multilistener.Listen(map[string][]string{
        "tcp":  {"127.0.0.1:8080"},
        "tcp6": {"[::1]:8080"},
    })

    if err != nil {
        log.Fatal("error when listening on valid addresses", err)
    }

    log.Printf("Listening on: %s - %s", m.Addr().Network(), m.Addr().String())

    for {
        c, err := m.Accept()
        if err != nil {
            log.Fatal("error when accepting from listeners", err)
        }

        log.Println(c.LocalAddr(), c.RemoteAddr())

        err = c.Close()
        if err != nil {
            log.Fatal("error when closing connection", err)
        }
    }
}
```

# gracecey

[![GoDoc](https://godoc.org/github.com/athiwatp/gracecey?status.svg)](https://godoc.org/github.com/athiwatp/gracecey)
[![Go Report Card](https://goreportcard.com/badge/github.com/athiwatp/gracecey)](https://goreportcard.com/report/github.com/athiwatp/gracecey)
[![Build Status](https://travis-ci.org/pseidemann/finish.svg?branch=master)](https://travis-ci.org/athiwatp/gracecey)

method and therefore requires Go 1.8+.

## Quick Start

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/athiwatp/gracecey"
)

func main() {
	http.HandleFunc("/helloworld", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "gracecey")
	})

	srv := &http.Server{Addr: "localhost:8080"}

	graceful := gracecey.New()
	graceful.Add(srv)

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	graceful.Wait()
}
```

Now execute that file:
```sh
$ go run example.go
```

Do a HTTP GET request:
```sh
$ curl localhost:8080/helloworld
```

This will print "world" after 5 seconds.

When the server is terminated with pressing `Ctrl+C` or `kill`, while `/hello` is
loading, finish will wait until the request was handled, before the server gets
killed.

The output will look like this:
```
2038/01/19 03:14:08 finish: shutdown signal received
2038/01/19 03:14:08 finish: shutting down server ...
2038/01/19 03:14:11 finish: server closed
```

### Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/athiwatp/gracecey"
	"github.com/sirupsen/logrus"
)

func main() {
	routerPub := httprouter.New()
	routerPub.HandlerFunc("GET", "/helloworld", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Second)
		fmt.Fprintln(w, "gracecey")
	})

	routerInt := httprouter.New()
	routerInt.HandlerFunc("GET", "/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	srvPub := &http.Server{Addr: "localhost:8080", Handler: routerPub}
	srvInt := &http.Server{Addr: "localhost:3000", Handler: routerInt}

	graceful := &gracecey.FlushFinish{
		Timeout: 60 * time.Second,
		Log:     logrus.StandardLogger(),
		Signals: append(gracecey.DefaultSignals, syscall.SIGHUP),
	}
	graceful.Add(srvPub, gracecey.WithName("the public server"))
	graceful.Add(srvInt, gracecey.WithName("the internal server"), gracecey.WithTimeout(60*time.Second))

	go func() {
		logrus.Infof("starting public server at %s", srvPub.Addr)
		err := srvPub.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		logrus.Infof("starting internal server at %s", srvInt.Addr)
		err := srvInt.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	graceful.Wait()
}
```

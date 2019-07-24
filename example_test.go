package gracecey_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/athiwatp/gracecey"
)

func Example() {
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

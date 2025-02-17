package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

type IdleTracker struct {
	active map[net.Conn]bool
	idle   time.Duration
	timer  *time.Timer
}

func NewIdleTracker(idle time.Duration) *IdleTracker {
	return &IdleTracker{
		active: make(map[net.Conn]bool),
		idle:   idle,
		timer:  time.NewTimer(idle),
	}
}

func (t *IdleTracker) Done() <-chan time.Time {
	return t.timer.C
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func main() {
	var host = flag.String("host", "http://localhost", "host")
	var port = flag.String("port", "8080", "port")
	var timeInSeconds = flag.Int("time", 60, "time in seconds")
	flag.Parse()

	remote, err := url.Parse(*host)
	if err != nil {
		panic(err)
	}

	idle := NewIdleTracker(time.Duration(*timeInSeconds) * time.Second)

	blacklist := []string{"/server/health", "/server/ping", "/admin/assets"}

	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL)

			if contains(blacklist, r.URL.String()) {
				//fmt.Println("blacklisted, not resetting URL")
			} else {
				idle.timer.Reset(idle.idle)
			}

			r.Host = remote.Host
			w.Header().Set("X-Ben", "Rad")
			p.ServeHTTP(w, r)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))

	fmt.Println("Starting server")

	go func() {
		<-idle.Done()
		fmt.Println("Shutting down server")
		os.Exit(0)
	}()

	err = http.ListenAndServe(fmt.Sprintf(":%s", *port), nil)
	if err != nil {
		panic(err)
	}
}

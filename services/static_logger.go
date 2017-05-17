package services

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

var (
	msgs  []string
	l     sync.RWMutex
	file  *os.File
	file2 *os.File
)

func init() {
	msgs = make([]string, 0, 20)
	Shout(fmt.Sprintf("cap %d", cap(msgs)))
	go func() {
		for range time.NewTicker(time.Minute).C {
			Dump()
		}
	}()
	Important("begin important log (server started?)")

	{
		f, e := os.Create("/tmp/important")
		if e != nil {
			Important("erorr opening file: " + e.Error())
		} else {
			file = f
		}
	}

	{
		f, e := os.Create("important")
		if e != nil {
			Important("erorr opening file: " + e.Error())
		} else {
			file2 = f
		}
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		s := <-c
		Important(fmt.Sprintf("got signal %d!", s))
		Dump()
		os.Exit(0)
	}()
}

func Important(s string) {
	l.Lock()
	defer l.Unlock()
	Shout(s)
	msgs = append(msgs[:0], append([]string{s}, msgs[0:]...)...)

	if len(msgs) > cap(msgs)-1 {
		Shout(fmt.Sprintf("shrinking from %d to %d", len(msgs), cap(msgs)-1))
		msgs = msgs[:cap(msgs)-1]
	}
}

func Shout(m string) {
	fmt.Fprintln(os.Stdout, "important! stdout: "+m)
	fmt.Fprintln(os.Stderr, "important! stderr: "+m)
	if file != nil {
		file.WriteString(m)
	}
	if file2 != nil {
		file2.WriteString(m)
	}
}

func Dump() {
	l.RLock()
	defer l.RUnlock()
	fmt.Println(msgs)
	m := strings.Join(msgs, "\r\n")
	Shout(m)
}

func Handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	l.RLock()
	defer l.RUnlock()
	m := strings.Join(msgs, "\r\n")
	w.Write([]byte(m))
}

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// aben <url> <payload_file> <count> <user> <password>

type Mark struct {
	StatusCode int
	Latency    time.Duration
}

func work(wg *sync.WaitGroup, url string, payload []byte, count int, mCh chan<- Mark) {
	wg.Add(1)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	reader := bytes.NewReader(payload)
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(os.Args[4], os.Args[5])
	go func() {
		defer wg.Done()
		for x := 0; x < count; x++ {
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				slog.Error("request failed", "error", err)
			} else {
				mCh <- Mark{resp.StatusCode, time.Now().Sub(start)}
				reader.Seek(0, 0)
			}
		}
	}()
}

func main() {
	url := os.Args[1]
	payload, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	count, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}

	wg := new(sync.WaitGroup)
	mCh := make(chan Mark, runtime.NumCPU())
	for x := 0; x < runtime.NumCPU(); x++ {
		work(wg, url, payload, count, mCh)
	}

	go func() {
		wg.Wait()
		close(mCh)
	}()

	var counter int
	for m := range mCh {
		counter++
		fmt.Printf("%d,%d,%d\n", counter, m.Latency.Milliseconds(), m.StatusCode)
	}
}

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type service struct {
	URL []string
	mux sync.RWMutex
}

func (s service) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html>
	<html>
		<head><title>MultiMetrics Exporter Proxy</title></head>
		<body>
			<h1>MultiMetrics Exporter Proxy</h1>
			<p><a href="/metrics">Metrics</a></p>
		</body>
	</html>`)
}

func fetch(url string, ch chan<- string) {
	// start := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		ch <- fmt.Sprintf("# URL=%s ERROR: %v\n", url, err) // send to channel ch
		return
	}
	defer resp.Body.Close() // don't leak resources

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ch <- fmt.Sprintf("## URL=%s READING ERROR: %v", url, err)
		return
	}
	// secs := time.Since(start).Seconds()
	ch <- string(bodyBytes)
}

func (s service) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// start := time.Now()
	ch := make(chan string)
	results := []string{}
	for _, url := range s.URL {
		go fetch(url, ch) // start a goroutine
	}
	for {
		result := <-ch // receive from channel ch
		results = append(results, result)
		if len(results) == len(s.URL) {
			break
		}
	}

	// fmt.Printf("## %.2fs total elapsed\n", time.Since(start).Seconds())
	io.WriteString(w, strings.Join(results, "\n"))
}

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		log.Fatal("FATAL: Arguments expected as URLs to be proxied")
	}
	s := service{
		URL: argsWithoutProg,
		mux: sync.RWMutex{},
	}

	http.HandleFunc("/metrics", s.metricsHandler)
	http.HandleFunc("/", s.indexHandler)

	port, okPort := os.LookupEnv("PORT")
	if !okPort {
		port = "9494"
	}
	host, okH := os.LookupEnv("HOST")
	if !okH {
		host = "0.0.0.0" // default is for docker
	}
	log.Println("Listening to " + host + ":" + port)
	for _, u := range s.URL {
		log.Println("URL: ", u)
	}
	log.Fatal(http.ListenAndServe(host+":"+port, nil))
}

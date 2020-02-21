package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Flags
var (
	successPattern = flag.String("success-pattern", "success|pass", "Regular expression to identify successes in the process out put.")
	errorPattern   = flag.String("error-pattern", "error|fail", "Regular expression to identify errors in the process out put.")
	metricsAddr    = flag.String("listen-address", ":9090", "The address to listen on for HTTP requests.")
)

// Metrics
var (
	successCounter = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "monitor", Name: "success", Help: "Counts the command successes"})
	errorCounter   = prometheus.NewCounter(prometheus.CounterOpts{Namespace: "monitor", Name: "error", Help: "Counts the command errors"})
)

func scan(reader io.Reader, lnHandler func(l string)) {
	s := bufio.NewScanner(reader)
	for s.Scan() {
		ln := s.Text()
		lnHandler(ln)
		fmt.Println(ln)
	}

	if err := s.Err(); err != nil {
		log.Printf("Scan error: %s", err.Error())
	}
}

func main() {
	flag.Parse()

	var exit = -1

	prometheus.MustRegister(successCounter)
	prometheus.MustRegister(errorCounter)

	successMatcher := regexp.MustCompile(*successPattern)
	errorMatcher := regexp.MustCompile(*errorPattern)

	outputHandler := func(s string) {
		if errorMatcher.MatchString(s) {
			errorCounter.Inc()
		} else if successMatcher.MatchString(s) {
			successCounter.Inc()
		}
	}

	var args strings.Builder
	for _, a := range flag.Args() {
		args.WriteString(" ")
		args.WriteString(strconv.Quote(a))
	}

	go func() {
		cmd := exec.Command("sh", "-c", args.String())
		cmd.Env = append(os.Environ())

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Unable to open stdout pipe: %s", err.Error())
		}
		go scan(stdout, outputHandler)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Unable to open stderr pipe: %s", err.Error())
		}
		go scan(stderr, outputHandler)

		err = cmd.Start()
		if err != nil {
			errorCounter.Inc()
			log.Println(err)
		}

		err = cmd.Wait()
		if err != nil {
			exit = 1

			if exerr, ok := err.(*exec.ExitError); ok {
				exit = exerr.ExitCode()
			}

			errorCounter.Inc()
			log.Println(err)
			return
		}

		successCounter.Inc()
		exit = 0
	}()

	flush := 0
	promhandler := promhttp.Handler()
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhandler.ServeHTTP(w, r)

		if exit != -1 {
			flush++
			if flush > 2 {
				os.Exit(exit)
			}
		}

	})
	log.Fatal(http.ListenAndServe(*metricsAddr, nil))
}

package main

import (
	"bufio"
	"flag"
	"fmt"
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
	delay          = flag.Int64("delay", 10, "Delay (in secods) between command executions.")
)

// Metrics
var (
	successCounter = prometheus.NewCounter(prometheus.CounterOpts{Name: "Success", Help: "Counts the successes"})
	errorCounter   = prometheus.NewCounter(prometheus.CounterOpts{Name: "Error", Help: "Counts the errors"})
)

func main() {
	flag.Parse()

	prometheus.MustRegister(successCounter)
	prometheus.MustRegister(errorCounter)

	successMatcher := regexp.MustCompile(*successPattern)
	errorMatcher := regexp.MustCompile(*errorPattern)

	var exit = -1

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
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Unable to open stderr pipe: %s", err.Error())
		}

		// stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if errorMatcher.MatchString(line) {
					errorCounter.Inc()
				} else if successMatcher.MatchString(line) {
					successCounter.Inc()
				}

				fmt.Println(line)
			}

			if err := scanner.Err(); err != nil {
				log.Printf("stdout scan error: %s", err.Error())
			}

			stdout.Close()
		}()

		// stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				if errorMatcher.MatchString(line) {
					errorCounter.Inc()
				} else if successMatcher.MatchString(line) {
					successCounter.Inc()
				}

				fmt.Println(line)
			}

			if err := scanner.Err(); err != nil {
				log.Printf("stderr scan error: %s", err.Error())
			}

			stderr.Close()
		}()

		err = cmd.Start()
		if err != nil {
			errorCounter.Inc()
			log.Println(err)
		}

		err = cmd.Wait()
		if err != nil {
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

	promhandler := promhttp.Handler()
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhandler.ServeHTTP(w, r)
		if exit != -1 {
			os.Exit(exit)
		}
	})

	log.Fatal(http.ListenAndServe(*metricsAddr, nil))
}

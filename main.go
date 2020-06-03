package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alandiegosantos/ab_testing/pkg/et"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

var (
	debugFlag   = flag.Bool("debug", false, "Enable debugging")
	versionFlag = flag.Bool("version", false, "Print version and stop")
	httpAddr    = flag.String("httpAddr", ":8080", "HTTP port")

	httpServer *http.Server
	isReady    bool = false
)

var (
	BuildVersion = "dev"
	BuildDate    = time.Now().Format("2006-01-02T15:04:05")
)

func main() {

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version: %s\nBuild on %s\n", BuildVersion, BuildDate)
		os.Exit(0)
	}

	if *debugFlag {
		// to change the flags on the default logger
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	ctx, cancel := context.WithCancel(context.Background())

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(interrupt)

	var g errgroup.Group

	g.Go(func() error {

		router := http.NewServeMux()

		router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}

			w.WriteHeader(http.StatusOK)

		})

		router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}

			if isReady {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}

		})

		router.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {

			tmpl, err := template.ParseFiles("./web/index.tmpl")
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// A sample config
			experiments := et.GetExperimentValues()

			// Setting cookies to track the experiments
			// Browser will take care of sending the cookies to another request
			for title, version := range experiments {
				cookie := &http.Cookie{
					Name:    fmt.Sprintf("experiment_%s", title),
					Value:   version,
					Expires: time.Now().Add(10 * time.Minute),
				}
				http.SetCookie(w, cookie)

			}

			err = tmpl.Execute(w, experiments)
			if err != nil {
				log.Print("execute: ", err)
				w.WriteHeader(http.StatusNotFound)
				return
			}

		})

		router.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {

			cookies := r.Cookies()

			for _, cookie := range cookies {

				if strings.HasPrefix(cookie.Name, "experiment_") {

					et.IncConversionCounter(cookie.Name[11:], cookie.Value)

				}

			}

			http.Redirect(w, r, "/index", http.StatusTemporaryRedirect)

		})

		// Redirect every other route to /index
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

			http.Redirect(w, r, "/index", http.StatusTemporaryRedirect)

		})

		router.HandleFunc("/favicon", func(w http.ResponseWriter, r *http.Request) {

			http.NotFound(w, r)

		})

		// Enable prometheus endpoint
		router.Handle("/metrics", promhttp.Handler())

		httpServer = &http.Server{
			Addr:         *httpAddr,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		}

		log.Printf("HTTP health server serving at %s", *httpAddr)

		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}

		return nil

	})

	select {
	case <-ctx.Done():
	case <-interrupt:
		break
	}

	log.Printf("Stopping web server")

	cancel()

	if httpServer != nil {
		httpServer.Shutdown(ctx)
	}

	if err := g.Wait(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(2)
	}

}

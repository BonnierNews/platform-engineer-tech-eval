package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/BonnierNews/platform-engineer-tech-eval/statik"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rakyll/statik/fs"
)

var (
	httpRequestsResponseTime prometheus.Summary
	httpRequestsTotal        *prometheus.CounterVec
	version                  prometheus.Gauge
	httpSizesTotal           *prometheus.CounterVec
)

func init() {
	httpRequestsResponseTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "example",
		Name:      "response_time_seconds",
		Help:      "Request response times",
	})
	version = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "example",
		Name:      "version",
		Help:      "Version information about this binary",
		ConstLabels: map[string]string{
			"version": "v0.1.0",
		},
	})
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "example",
		Name:      "requests_total",
		Help:      "Count of all HTTP requests",
	}, []string{"code", "method"})

	httpSizesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "example",
		Name:      "size_by_path_total",
		Help:      "Count of size sent by path",
	}, []string{"path", "method"})

	prometheus.MustRegister(httpRequestsResponseTime, version, httpRequestsTotal, httpSizesTotal)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		httpRequestsResponseTime.Observe(float64(time.Since(start).Seconds()))
		s, _ := strconv.ParseFloat(w.Header().Get("Content-Length"), 64)
		httpSizesTotal.With(prometheus.Labels{"path": r.RequestURI, "method": r.Method}).Add(s)
	})
}

func doProxy(w http.ResponseWriter, r *http.Request) {
	u, _ := url.Parse(os.Getenv("PROXY_HOST"))
	r.URL.Host = u.Host
	r.Host = u.Host
	r.URL.Scheme = u.Scheme
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(w, r)

}

func main() {
	bind := ""
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&bind, "bind", ":8080", "The socket to bind to.")
	flagset.Parse(os.Args[1:])

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.RawQuery
		if q == "proxy=true" {
			doProxy(w, r)
			return
		}
		fmt.Printf("%v", q)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from example application. Try '/image.jpg' for more fun...\n"))
	})

	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.RawQuery
		if q == "proxy=true" {
			doProxy(w, r)
			return
		}
		f, err := statikFS.Open("/image.jpg")
		if err != nil {
			fmt.Printf("%v", err)
			http.NotFound(w, r)
			return
		}
		fHeader := make([]byte, 512)
		f.Read(fHeader)
		fContentType := http.DetectContentType(fHeader)
		fStat, _ := f.Stat()
		fSize := strconv.FormatInt(fStat.Size(), 10)
		w.Header().Set("Content-Type", fContentType)
		w.Header().Set("Content-Length", fSize)
		f.Seek(0, 0)
		io.Copy(w, f)
	})

	handler := http.NewServeMux()
	handler.Handle("/", promhttp.InstrumentHandlerCounter(httpRequestsTotal, rootHandler))
	fmt.Printf("Registering / handler\n")
	handler.Handle("/image.jpg", promhttp.InstrumentHandlerCounter(httpRequestsTotal, fileHandler))
	fmt.Printf("Registering /image.jpg handler\n")
	handler.Handle("/metrics", promhttp.Handler())
	metrics := middleware(handler)
	fmt.Printf("Starting server on %s\n", bind)
	log.Fatal(http.ListenAndServe(bind, metrics))
}

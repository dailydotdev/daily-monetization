package main

import (
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/api/option"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var gcpOpts []option.ClientOption
var campaignsCount, _ = strconv.Atoi(os.Getenv("CAMPAIGNS_COUNT"))

func ServeAd(w http.ResponseWriter, r *http.Request) {
	var res []interface{}

	cf, err := fetchCodefund(r, "a4ace977-6531-4708-a4d9-413c8910ac2c")
	if err != nil {
		log.Warn("failed to fetch ad from Codefund ", err)
	} else if cf != nil {
		res = []interface{}{*cf}
	}

	if res == nil {
		bsa, err := fetchBsa(r)
		if err != nil {
			log.Warn("failed to fetch ad from BSA ", err)
		} else if bsa != nil {
			res = []interface{}{*bsa}
		}
	}
	if res == nil {
		camps, err := fetchCampaigns(r.Context(), time.Now())
		if err != nil {
			log.Warn("failed to fetch campaigns ", err)
		} else if camps != nil {
			index := rand.Intn(len(camps))
			res = []interface{}{camps[index]}
		}
	}

	if res == nil {
		log.Info("no ads to serve for extension")
		res = []interface{}{}
	}

	js, err := marshalJSON(res)
	if err != nil {
		log.Error("failed to marshal json ", err)
		http.Error(w, "Server Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func ServeToilet(w http.ResponseWriter, r *http.Request) {
	var res []interface{}

	cf, err := fetchCodefund(r, "89dc8cbd-475f-4941-bfa8-03e509b8f897")
	if err != nil {
		log.Warn("failed to fetch ad from Codefund ", err)
	} else if cf != nil {
		res = []interface{}{*cf}
	}

	if res == nil {
		bsa, err := fetchBsa(r)
		if err != nil {
			log.Warn("failed to fetch ad from BSA ", err)
		} else if bsa != nil {
			res = []interface{}{*bsa}
		}
	}

	if res == nil {
		log.Info("no ads to serve for toilet")
		res = []interface{}{}
	}

	js, err := marshalJSON(res)
	if err != nil {
		log.Error("failed to marshal json ", err)
		http.Error(w, "Server Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func ServeBsa(w http.ResponseWriter, r *http.Request) {
	res, err := sendBsaRequest(r)
	if err != nil {
		log.Warn("failed to fetch ad from BSA ", err)
		http.Error(w, "Server Internal Error", http.StatusInternalServerError)
		return
	}

	js, err := marshalJSON(res.Ads)
	if err != nil {
		log.Error("failed to marshal json ", err)
		http.Error(w, "Server Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

type HealthHandler struct{}
type AdsHandler struct{}
type App struct {
	HealthHandler *HealthHandler
	AdsHandler    *AdsHandler
}

func (h *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string
	head, r.URL.Path = shiftPath(r.URL.Path)

	switch head {
	case "health":
		h.HealthHandler.ServeHTTP(w, r)
		return
	case "a":
		h.AdsHandler.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *AdsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if r.URL.Path == "/" {
			ServeAd(w, r)
			return
		}

		if r.URL.Path == "/toilet" {
			ServeToilet(w, r)
			return
		}

		_, tail := shiftPath(r.URL.Path)
		if tail == "/" {
			ServeBsa(w, r)
			return
		}
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" && r.Method == "GET" {
		fmt.Fprintf(w, "OK")
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func createApp() *App {
	return &App{
		HealthHandler: new(HealthHandler),
		AdsHandler:    new(AdsHandler),
	}
}

func init() {
	hystrix.ConfigureCommand(hystrixDb, hystrix.CommandConfig{Timeout: 300, MaxConcurrentRequests: 100})
	hystrix.ConfigureCommand(hystrixCf, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})
	hystrix.ConfigureCommand(hystrixBsa, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})

	if file, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
		gcpOpts = append(gcpOpts, option.WithCredentialsFile(file))
	}

	log.SetOutput(os.Stdout)
	if getEnv("ENV", "DEV") == "PROD" {
		log.SetFormatter(&log.JSONFormatter{})

		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:          os.Getenv("GCLOUD_PROJECT"),
			TraceClientOptions: gcpOpts,
		})
		if err != nil {
			log.Fatal(err)
		}
		trace.RegisterExporter(exporter)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

		httpClient = &http.Client{
			Transport: &ochttp.Transport{
				// Use Google Cloud propagation format.
				Propagation: &propagation.HTTPFormat{},
			},
		}
	} else {
		httpClient = &http.Client{}
	}

	err := configurePubsub()
	if err != nil {
		log.Fatal("failed to initialize google pub/sub client ", err)
	}
}

func main() {
	migrateDatabase()

	initializeDatabase()
	defer tearDatabase()

	go subscribeToNewAd()

	app := createApp()
	addr := fmt.Sprintf(":%s", getEnv("PORT", "9090"))
	log.Info("server is listening to ", addr)
	err := http.ListenAndServe(addr, &ochttp.Handler{Handler: app, Propagation: &propagation.HTTPFormat{}}) // set listen addr
	if err != nil {
		log.Fatal("failed to start listening ", err)
	}
}

package main

import (
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"google.golang.org/api/option"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Ad struct {
	Description string
	Image       string
	Link        string
	Source      string
}

var gcpOpts []option.ClientOption
var campaignsCount, _ = strconv.Atoi(os.Getenv("CAMPAIGNS_COUNT"))

func Health(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "OK")
}

func ServeAd(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []interface{}

	camps, err := fetchCampaigns(r.Context(), time.Now())
	if err != nil {
		log.Warn("failed to fetch campaigns ", err)
	} else if camps != nil {
		index := rand.Intn(campaignsCount)
		if index < len(camps) {
			res = []interface{}{camps[index]}
		}
	}

	if res == nil {
		cf, err := fetchCodefund(r, "a4ace977-6531-4708-a4d9-413c8910ac2c")
		if err != nil {
			log.Warn("failed to fetch ad from Codefund ", err)
		} else if cf != nil {
			res = []interface{}{*cf}
		}
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
		res = []interface{}{}
	}

	js, err := MarshalJSON(res)
	if err != nil {
		log.Error("failed to marshal json ", err)
		http.Error(w, "Server Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func createRouter() *httprouter.Router {
	router := httprouter.New()

	router.GET("/health", Health)
	router.GET("/a", ServeAd)
	return router
}

func init() {
	hystrix.ConfigureCommand(hystrixDb, hystrix.CommandConfig{Timeout: 300})
	hystrix.ConfigureCommand(hystrixCf, hystrix.CommandConfig{Timeout: 600})
	hystrix.ConfigureCommand(hystrixBsa, hystrix.CommandConfig{Timeout: 600})

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
}

func main() {
	migrateDatabase()

	initializeDatabase()
	defer tearDatabase()

	router := createRouter()
	addr := fmt.Sprintf(":%s", getEnv("PORT", "9090"))
	log.Info("server is listening to ", addr)
	err := http.ListenAndServe(addr, &ochttp.Handler{Handler: router, Propagation: &propagation.HTTPFormat{}}) // set listen addr
	if err != nil {
		log.Fatal("failed to start listening ", err)
	}
}

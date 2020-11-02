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
	"strings"
	"time"
)

var gcpOpts []option.ClientOption
var segmentToId map[string]string = map[string]string{
	"frontend": "CE7I5K3Y",
	"backend":  "CE7I5K37",
	"devops":   "CE7I5KQE",
	"":         "CK7DT2QM",
}

func getBsaAd(r *http.Request, country string, segment string) (*BsaAd, error) {
	var bsa *BsaAd
	var err error
	if country == "united states" {
		bsa, err = fetchBsa(r, "CE7D5KJL")
	} else {
		bsa, err = fetchBsa(r, segmentToId[segment])
	}
	if err != nil {
		log.Warn("failed to fetch ad from BSA ", err)
	}
	return bsa, err
}

func ServeAd(w http.ResponseWriter, r *http.Request) {
	var res []interface{}

	ip := getIpAddress(r)
	country := getCountryByIP(ip)

	camps, err := fetchCampaigns(r.Context(), time.Now())
	if err != nil {
		log.Warn("failed to fetch campaigns ", err)
	}

	// Look for a campaign ad based on probability
	prob := rand.Float32()
	for i := 0; i < len(camps); i++ {
		if !camps[i].Fallback && (len(camps[i].Geo) == 0 || strings.Contains(camps[i].Geo, country)) {
			if prob <= camps[i].Probability {
				res = []interface{}{camps[i]}
				break
			}
			prob -= camps[i].Probability
		}
	}

	var userId string
	cookie, _ := r.Cookie("da2")
	if cookie != nil {
		userId = cookie.Value
	}
	segment, _ := findSegment(r.Context(), userId)
	prob = rand.Float32()
	if prob < 0.1 {
		if res == nil {
			cf, err := fetchEthicalAds(r, segment)
			if err != nil {
				log.Warn("failed to fetch ad from EthicalAds ", err)
			} else if cf != nil {
				res = []interface{}{*cf}
			}
		}
		if res == nil {
			bsa, _ := getBsaAd(r, country, segment)
			if bsa != nil {
				res = []interface{}{*bsa}
			}
		}
	} else {
		if res == nil {
			bsa, _ := getBsaAd(r, country, segment)
			if bsa != nil {
				res = []interface{}{*bsa}
			}
		}
		if res == nil {
			cf, err := fetchEthicalAds(r, segment)
			if err != nil {
				log.Warn("failed to fetch ad from EthicalAds ", err)
			} else if cf != nil {
				res = []interface{}{*cf}
			}
		}
	}

	if res == nil {
		// Look for a fallback campaign ad based on probability
		prob := rand.Float32()
		for i := 0; i < len(camps); i++ {
			if camps[i].Fallback && (len(camps[i].Geo) == 0 || strings.Contains(country, camps[i].Geo)) {
				if prob <= camps[i].Probability {
					res = []interface{}{camps[i]}
					break
				}
				prob -= camps[i].Probability
			}
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

	bsa, err := fetchBsa(r, "CK7DT2QM")
	if err != nil {
		log.Warn("failed to fetch ad from BSA ", err)
	} else if bsa != nil {
		res = []interface{}{*bsa}
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
	res, err := sendBsaRequest(r, "CK7DT2QM")
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
	case "v1":
		head, r.URL.Path = shiftPath(r.URL.Path)
		if head == "a" {
			h.AdsHandler.ServeHTTP(w, r)
		}
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
	hystrix.ConfigureCommand(hystrixBsa, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})
	hystrix.ConfigureCommand(hystrixEa, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})

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

// 	err := configurePubsub()
// 	if err != nil {
// 		log.Fatal("failed to initialize google pub/sub client ", err)
// 	}
}

func main() {
	openGeolocationDatabase()
	defer closeGeolocationDatabase()

	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()

	go subscribeToNewAd()
	go subscribeToSegmentFound()

	app := createApp()
	addr := fmt.Sprintf(":%s", getEnv("PORT", "9090"))
	log.Info("server is listening to ", addr)
	err := http.ListenAndServe(addr, &ochttp.Handler{Handler: app, Propagation: &propagation.HTTPFormat{}}) // set listen addr
	if err != nil {
		log.Fatal("failed to start listening ", err)
	}
}

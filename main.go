package main

import (
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"encoding/json"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/api/option"
	"io/ioutil"
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

func segmentToThresholds(segment string) float32 {
	if segment == "devops" {
		return 0.5
	}
	return 0.1
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

	// Premium self-serve
	if res == nil {
		bsa, err := fetchBsa(r, "CEBI62JM")
		if err != nil {
			log.Warn("failed to fetch ad from premium self-serve ", err)
		} else if bsa != nil {
			res = []interface{}{*bsa}
		}
	}

	var userId string
	cookie, _ := r.Cookie("da2")
	if cookie != nil {
		userId = cookie.Value
	}
	segment, _ := findSegment(r.Context(), userId)
	prob = rand.Float32()
	threshold := segmentToThresholds(segment)
	if prob < threshold {
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

	// Standard self-serve
	if res == nil {
		bsa, err := fetchBsa(r, "CEBI62J7")
		if err != nil {
			log.Warn("failed to fetch ad from standard self-serve ", err)
		} else if bsa != nil {
			res = []interface{}{*bsa}
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

type PubSubMessage struct {
	Message struct {
		Data []byte `json:"data,omitempty"`
		ID   string `json:"id"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type HealthHandler struct{}
type AdsHandler struct{}
type App struct {
	HealthHandler *HealthHandler
	AdsHandler    *AdsHandler
}

type NewAdHandler struct{}
type SegmentFoundHandler struct{}
type BackgroundApp struct {
	HealthHandler       *HealthHandler
	NewAdHandler        *NewAdHandler
	SegmentFoundHandler *SegmentFoundHandler
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

func (h *BackgroundApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string
	head, r.URL.Path = shiftPath(r.URL.Path)

	switch head {
	case "health":
		h.HealthHandler.ServeHTTP(w, r)
		return
	case "newAd":
		h.NewAdHandler.ServeHTTP(w, r)
		return
	case "segmentFound":
		h.SegmentFoundHandler.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (h *NewAdHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.URL.Path == "/" {
			var msg PubSubMessage
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("ioutil.ReadAll: %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(body, &msg); err != nil {
				log.Printf("json.Unmarshal: %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			var ad ScheduledCampaignAd
			if err := json.Unmarshal(msg.Message.Data, &ad); err != nil {
				log.Errorf("failed to decode message %v", err)
				return
			}

			log.Infof("[AD %s] adding new campaign ad", ad.Id)
			if err := addCampaign(r.Context(), ad); err != nil {
				log.WithField("ad", ad).Errorf("[AD %s] failed to add new campaign ad %v", ad.Id, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			log.Infof("[AD %s] added new campaign ad", ad.Id)
			return
		}
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

type SegmentMessage struct {
	UserId  string
	Segment string
}

func (h *SegmentFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.URL.Path == "/" {
			var msg PubSubMessage
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("ioutil.ReadAll: %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(body, &msg); err != nil {
				log.Printf("json.Unmarshal: %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			var data SegmentMessage
			if err := json.Unmarshal(msg.Message.Data, &data); err != nil {
				log.Errorf("failed to decode message %v", err)
				return
			}

			if err := updateUserSegment(r.Context(), data.UserId, data.Segment); err != nil {
				log.WithField("segment", data).Errorf("failed to update user segment %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			log.WithField("segment", data).Infof("updated user segment")
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

func createBackgroundApp() *BackgroundApp {
	return &BackgroundApp{
		HealthHandler: new(HealthHandler),
		NewAdHandler:  new(NewAdHandler),
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
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(0.25)})

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
	openGeolocationDatabase()
	defer closeGeolocationDatabase()

	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()

	var app http.Handler
	if len(os.Args) > 1 && os.Args[1] == "background" {
		log.Info("background processing is on")
		app = createBackgroundApp()
	} else {
		app = createApp()
	}
	addr := fmt.Sprintf(":%s", getEnv("PORT", "9090"))
	log.Info("server is listening to ", addr)
	err := http.ListenAndServe(addr, &ochttp.Handler{Handler: app, Propagation: &propagation.HTTPFormat{}}) // set listen addr
	if err != nil {
		log.Fatal("failed to start listening ", err)
	}
}

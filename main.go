package main

import (
	"cloud.google.com/go/pubsub"
	"context"
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
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var gcpOpts []option.ClientOption
var segmentToId map[string]string = map[string]string{
	"frontend":   "CE7I5K3Y",
	"backend":    "CE7I5K37",
	"devops":     "CE7I5KQE",
	"kubernetes": "CE7I5KQE",
	"crypto":     "CK7DT2QM",
	"":           "CK7DT2QM",
}
var pubsubClient *pubsub.Client = nil

func segmentToThresholds(segment string) float32 {
	//if segment == "devops" {
	//	return 1
	//}
	return 0.2
}

func getBsaAd(r *http.Request, country string, segment string, active bool) (*BsaAd, error) {
	var bsa *BsaAd
	var err error
	if active {
		bsa, err = fetchBsa(r, "CEAIP23E")
	} else if country == "united states" {
		bsa, err = fetchBsa(r, "CE7D5KJL")
	} else if country == "united kingdom" {
		bsa, err = fetchBsa(r, "CEAD62QI")
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
	active := r.URL.Query().Get("active") == "true"
	var segment = ""
	var userId string
	cookie, _ := r.Cookie("da2")
	if cookie != nil {
		userId = cookie.Value
	}

	if country == "united states" {
		bsa, _ := getBsaAd(r, country, segment, active)
		if bsa != nil {
			res = []interface{}{*bsa}
		}
	}

	camps, err := fetchCampaigns(r.Context(), time.Now(), userId)
	if err != nil {
		log.Warn("failed to fetch campaigns ", err)
	}

	if res == nil {

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
	}

	// Premium self-serve
	if res == nil {
		bsa, err := fetchBsa(r, "CEBI62JM")
		if err != nil {
			log.Warn("failed to fetch ad from premium self-serve ", err)
		} else if bsa != nil {
			bsa.ProviderId = "premium"
			res = []interface{}{*bsa}
		}
	}

	prob := rand.Float32()
	threshold := segmentToThresholds(segment)
	tags, err := getUserTags(r.Context(), userId)
	if err != nil {
		log.Warnln("getUserTags", err)
	}
	if prob < threshold {
		if res == nil {
			cf, err := fetchEthicalAds(r, tags)
			if err != nil {
				log.Warn("failed to fetch ad from EthicalAds ", err)
			} else if cf != nil {
				res = []interface{}{*cf}
			}
		}
		if res == nil {
			bsa, _ := getBsaAd(r, country, segment, active)
			if bsa != nil {
				res = []interface{}{*bsa}
			}
		}
	} else {
		if res == nil {
			bsa, _ := getBsaAd(r, country, segment, active)
			if bsa != nil {
				res = []interface{}{*bsa}
			}
		}
		if res == nil {
			cf, err := fetchEthicalAds(r, tags)
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
			bsa.ProviderId = "standard"
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

func NewAd(ctx context.Context, log *log.Entry, ad ScheduledCampaignAd) error {
	log.Infof("[AD %s] adding new campaign ad", ad.Id)
	if err := addCampaign(ctx, ad); err != nil {
		log.WithField("ad", ad).Errorf("[AD %s] failed to add new campaign ad %v", ad.Id, err)
		return err
	}

	log.Infof("[AD %s] added new campaign ad", ad.Id)
	return nil
}

type ViewMessage struct {
	UserId string
	Tags   []string
}

func View(ctx context.Context, log *log.Entry, data ViewMessage) error {
	if len(data.Tags) > 0 {
		if err := addOrUpdateUserTags(ctx, data.UserId, data.Tags); err != nil {
			log.WithField("view", data).Errorf("addOrUpdateUserTags %v", err)
			return err
		}
	}
	return nil
}

func DeleteOldTags(ctx context.Context, log *log.Entry) error {
	if err := deleteOldTags(ctx); err != nil {
		log.Errorf("deleteOldTags %v", err)
		return err
	}
	return nil
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

func subscribeToNewAd() {
	const sub = "monetization-new-ad"
	log.Info("receiving messages from ", sub)
	ctx := context.Background()
	err := pubsubClient.Subscription(sub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		childLog := log.WithField("messageId", msg.ID)
		var data ScheduledCampaignAd
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			childLog.Errorf("failed to decode message %v", err)
			msg.Ack()
			return
		}

		if err := NewAd(ctx, childLog, data); err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})

	if err != nil {
		log.Fatal("failed to receive messages from pubsub ", err)
	}
}

func subscribeToView() {
	const sub = "monetization-views"
	log.Info("receiving messages from ", sub)
	ctx := context.Background()
	err := pubsubClient.Subscription(sub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		childLog := log.WithField("messageId", msg.ID)
		var data ViewMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			childLog.Errorf("failed to decode message %v", err)
			msg.Ack()
			return
		}

		if err := View(ctx, childLog, data); err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})

	if err != nil {
		log.Fatal("failed to receive messages from pubsub ", err)
	}
}

func subscribeToDeleteOldTags() {
	const sub = "monetization-delete-old-tags"
	log.Info("receiving messages from ", sub)
	ctx := context.Background()
	err := pubsubClient.Subscription(sub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		childLog := log.WithField("messageId", msg.ID)
		if err := DeleteOldTags(ctx, childLog); err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})

	if err != nil {
		log.Fatal("failed to receive messages from pubsub ", err)
	}
}

func createBackgroundApp() {
	go subscribeToNewAd()
	go subscribeToView()
	subscribeToDeleteOldTags()
}

func init() {
	hystrix.ConfigureCommand(hystrixDb, hystrix.CommandConfig{Timeout: 300, MaxConcurrentRequests: 100})
	hystrix.ConfigureCommand(hystrixBsa, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})
	hystrix.ConfigureCommand(hystrixEa, hystrix.CommandConfig{Timeout: 700, MaxConcurrentRequests: 100})

	if file, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
		gcpOpts = append(gcpOpts, option.WithCredentialsFile(file))
	}

	projectID := os.Getenv("GCLOUD_PROJECT")
	ctx := context.Background()

	log.SetOutput(os.Stdout)
	if getEnv("ENV", "DEV") == "PROD" {
		log.SetFormatter(&log.JSONFormatter{})

		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:          projectID,
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

	var err error
	pubsubClient, err = pubsub.NewClient(ctx, projectID, gcpOpts...)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrateDatabase()
	} else {
		openGeolocationDatabase()
		defer closeGeolocationDatabase()

		initializeDatabase()
		defer tearDatabase()

		if len(os.Args) > 1 && os.Args[1] == "background" {
			log.Info("background processing is on")
			createBackgroundApp()
		} else {
			app := createApp()
			addr := fmt.Sprintf(":%s", getEnv("PORT", "9090"))
			log.Info("server is listening to ", addr)
			err := http.ListenAndServe(addr, &ochttp.Handler{Handler: app, Propagation: &propagation.HTTPFormat{}}) // set listen addr
			if err != nil {
				log.Fatal("failed to start listening ", err)
			}
		}
	}
}

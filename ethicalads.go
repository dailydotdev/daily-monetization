package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

type EthicalAdsAd struct {
	Ad
	Pixel        []string
	ReferralLink string
}

type EthicalAdsResponse struct {
	Id      string
	Body    string
	Image   string
	Link    string
	ViewUrl string `json:"view_url"`
	Nonce   string
}

var hystrixEa = "EthicalAds"
var ethicaladsToken = os.Getenv("ETHICALADS_TOKEN")

var fetchEthicalAds = func(r *http.Request, keywords []string) (*EthicalAdsAd, error) {
	keywordsString := ""
	for i, keyword := range keywords {
		if i > 0 {
			keywordsString += ", "
		}
		keywordsString += fmt.Sprintf("\"%s\"", keyword)
	}
	ip := getIpAddress(r)
	ua := r.UserAgent()
	var body = []byte(`{ "publisher": "dailydev", "placements": [{ "div_id": "ad-div-1", "ad_type": "image-v1" }], "campaign_types": ["paid"], "user_ip": "` + ip + `", "user_ua": "` + ua + `", "keywords": [` + keywordsString + `] }`)
	var res EthicalAdsResponse
	req, _ := http.NewRequest("POST", "https://server.ethicalads.io/api/v1/decision/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+ethicaladsToken)
	req = req.WithContext(r.Context())
	err := getJsonHystrix(hystrixEa, req, &res, true)
	if err != nil {
		return nil, err
	}
	if res.Body == "" {
		return nil, nil
	}

	ad := EthicalAdsAd{}
	ad.Company = "EthicalAds"
	ad.Description = res.Body
	ad.Link = res.Link
	ad.Source = "EthicalAds"
	ad.Pixel = []string{res.ViewUrl}
	ad.Image = res.Image
	ad.ReferralLink = "https://www.ethicalads.io/?ref=dailydev"
	ad.ProviderId = "ethical"

	return &ad, nil
}

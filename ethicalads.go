package main

import (
	"net/http"
	"net/url"
)

type EthicalAdsAd struct {
	Ad
	Pixel []string
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

var fetchEthicalAds = func(r *http.Request) (*EthicalAdsAd, error) {
	params := url.Values{}
	params.Add("callback", "ethicalads")
	params.Add("format", "json")
	params.Add("publisher", "dailydev")
	params.Add("div_ids", "sample-ad")
	params.Add("ad_types", "image-v1")
	params.Add("user_ip", getIpAddress(r))
	params.Add("user_ua", r.UserAgent())
	var res EthicalAdsResponse
	req, _ := http.NewRequest("GET", "https://server.ethicalads.io/api/v1/decision/?"+params.Encode(), nil)
	req = req.WithContext(r.Context())
	err := getJsonHystrix(hystrixEa, req, &res, true)

	if err != nil {
		return nil, err
	}

	ad := EthicalAdsAd{}
	ad.Company = "EthicalAds"
	ad.Description = res.Body
	ad.Link = res.Link
	ad.Source = "EthicalAds"
	ad.Pixel = []string{res.ViewUrl}
	ad.Image = res.Image

	return &ad, nil
}

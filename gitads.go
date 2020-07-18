package main

import (
	"net/http"
)

type GitAdsAd struct {
	Ad
}

type GitAdsResponse struct {
	Redirect string
	Image    string
	Text     string
}

var hystrixGa = "GitAds"

var fetchGitAds = func(r *http.Request) (*GitAdsAd, error) {
	var res GitAdsResponse
	req, _ := http.NewRequest("GET", "https://tracking.gitads.io/dailydev.json", nil)
	req = req.WithContext(r.Context())
	err := getJsonHystrix(hystrixGa, req, &res, true)
	if err != nil {
		return nil, err
	}
	if res.Text == "" {
		return nil, nil
	}

	ad := GitAdsAd{}
	ad.Company = "GitAds"
	ad.Description = res.Text
	ad.Link = res.Redirect
	ad.Source = "GitAds"
	ad.Image = res.Image

	return &ad, nil
}

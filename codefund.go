package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type CodefundAd struct {
	Ad
	Pixel        []string
	ReferralLink string
}

type CodefundImage struct {
	Format string
	Url    string
}

type CodefundRequest struct {
	Ip        string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

type CodefundResponse struct {
	CampaignUrl   string
	Headline      string
	Body          string
	ImpressionUrl string
	Images        []CodefundImage
	Fallback      bool
}

var hystrixCf = "Codefund"
var cfApiKey = os.Getenv("CODEFUND_API_KEY")
var referralLink = getEnv("CODEFUND_REFERRAL_LINK", "")

var fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
	var res CodefundResponse
	ip := getIpAddress(r)
	body, err := json.Marshal(CodefundRequest{Ip: ip, UserAgent: r.UserAgent()})
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", "https://app.codefund.io/properties/"+propertyId+"/funder.json", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", r.UserAgent())
	req.Header.Set("X-CodeFund-API-Key", cfApiKey)
	req = req.WithContext(r.Context())
	err = getJsonHystrix(hystrixCf, req, &res, true)

	if err != nil {
		return nil, err
	}

	if res.Fallback || res.CampaignUrl == "" {
		return nil, nil
	}

	ad := CodefundAd{}
	ad.Company = "CodeFund"
	ad.Description = res.Headline + " " + res.Body
	ad.Link = res.CampaignUrl
	ad.Source = ad.Company
	ad.ReferralLink = referralLink
	ad.Pixel = []string{res.ImpressionUrl}

	for _, image := range res.Images {
		if image.Format == "wide" {
			ad.Image = image.Url
			break
		}
	}
	if len(ad.Image) == 0 {
		for _, image := range res.Images {
			if image.Format == "large" {
				ad.Image = image.Url
				break
			}
		}
	}

	return &ad, nil
}

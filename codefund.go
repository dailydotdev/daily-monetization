package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type CodefundAd struct {
	Ad
	Company string
	Pixel   []string
}

type CodefundImage struct {
	SizeDescriptor string `json:"size_descriptor"`
	Url            string
}

type CodefundRequest struct {
	Ip        string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

type CodefundResponse struct {
	HouseAd       bool `json:"house_ad"`
	Link          string
	Headline      string
	Description   string
	LargeImageUrl string `json:"large_image_url"`
	Pixel         string
	Images        []CodefundImage
	Reason        string
}

var hystrixCf = "Codefund"
var cfApiKey = os.Getenv("CODEFUND_API_KEY")

var fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
	var res CodefundResponse
	ip := getIpAddress(r)
	body, err := json.Marshal(CodefundRequest{Ip: ip, UserAgent: r.UserAgent()})
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("POST", "https://codefund.io/api/v1/impression/"+propertyId, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", r.UserAgent())
	req.Header.Set("X-CodeFund-API-Key", cfApiKey)
	req = req.WithContext(r.Context())
	err = getJsonHystrix(hystrixCf, req, &res)

	if err != nil {
		return nil, err
	}

	if len(res.Reason) > 0 {
		return nil, nil
	}

	ad := CodefundAd{Company: "CodeFund"}
	ad.Description = res.Headline + " " + res.Description
	ad.Link = res.Link
	ad.Source = ad.Company

	for _, image := range res.Images {
		if image.SizeDescriptor == "wide" {
			ad.Image = image.Url
			break
		}
	}
	if len(ad.Image) == 0 {
		ad.Image = res.LargeImageUrl
	}

	if strings.HasPrefix(res.Pixel, "//") {
		ad.Pixel = []string{"https:" + res.Pixel}
	} else {
		ad.Pixel = []string{res.Pixel}
	}

	return &ad, nil
}

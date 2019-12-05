package main

import (
	"fmt"
	"net/http"
	"strings"
)

type BsaAd struct {
	Ad
	Pixel        []string
	ReferralLink string
}

type BsaResponse struct {
	Ads []map[string]interface{}
}

var hystrixBsa = "BSA"

func sendBsaRequest(r *http.Request, propertyId string) (BsaResponse, error) {
	var res BsaResponse
	ip := getIpAddress(r)
	//ip = "208.98.185.89"
	req, _ := http.NewRequest("GET", "https://srv.buysellads.com/ads/"+propertyId+".json?segment=placement:dailynowco&forwardedip="+ip, nil)
	req = req.WithContext(r.Context())

	err := getJsonHystrix(hystrixBsa, req, &res, false)
	if err != nil {
		return BsaResponse{}, err
	}

	return res, nil
}

var fetchBsa = func(r *http.Request, propertyId string) (*BsaAd, error) {
	res, err := sendBsaRequest(r, propertyId)
	if err != nil {
		return nil, err
	}

	ads := res.Ads
	for _, ad := range ads {
		if _, ok := ad["statlink"]; ok {
			retAd := BsaAd{}
			retAd.Description, _ = ad["title"].(string)
			retAd.Image, _ = ad["smallImage"].(string)
			retAd.Link, _ = ad["statlink"].(string)
			retAd.Link = fmt.Sprintf("https:%s", retAd.Link)
			retAd.ReferralLink, _ = ad["ad_via_link"].(string)
			retAd.Source = "Carbon"
			retAd.Company = retAd.Source
			if pixel, ok := ad["pixel"].(string); ok {
				retAd.Pixel = strings.Split(pixel, "||")
				for index := range retAd.Pixel {
					retAd.Pixel[index] = strings.Replace(retAd.Pixel[index], "[timestamp]", ad["timestamp"].(string), -1)
				}
			} else {
				retAd.Pixel = []string{}
			}
			return &retAd, nil
		}
	}

	return nil, nil
}

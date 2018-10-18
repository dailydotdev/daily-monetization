package main

import (
	"fmt"
	"net/http"
	"strings"
)

type BsaAd struct {
	Ad
	Company         string
	Pixel           []string
	BackgroundColor string
}

type BsaResponse struct {
	Ads []map[string]interface{}
}

var hystrixBsa = "BSA"

var fetchBsa = func(r *http.Request) (*BsaAd, error) {
	var res BsaResponse
	ip := r.RemoteAddr
	//ip = "208.98.185.89"
	req, _ := http.NewRequest("GET", "https://srv.buysellads.com/ads/CKYI623Y.json?segment=placement:dailynowco&forwardedip="+ip, nil)
	req = req.WithContext(r.Context())

	err := getJsonHystrix(hystrixBsa, req, &res)
	if err != nil {
		return nil, err
	}

	ads := res.Ads
	for _, ad := range ads {
		if _, ok := ad["statlink"]; ok {
			retAd := BsaAd{}
			retAd.Company, _ = ad["company"].(string)
			retAd.Description, _ = ad["description"].(string)
			retAd.Image, _ = ad["logo"].(string)
			retAd.Link, _ = ad["statlink"].(string)
			retAd.Link = fmt.Sprintf("https:%s", retAd.Link)
			retAd.BackgroundColor, _ = ad["backgroundColor"].(string)
			retAd.Source = "BSA"
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

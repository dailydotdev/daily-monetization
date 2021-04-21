package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var ad = Ad{
	Source:      "source",
	Image:       "image",
	Link:        "http://link.com",
	Description: "desc",
	Company:     "company",
}

var campaignNotAvailable = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
	return nil, nil
}

var bsaNotAvailable = func(r *http.Request, propertyId string) (*BsaAd, error) {
	return nil, nil
}

var emptySegment = func(ctx context.Context, userId string) (string, error) {
	return "", nil
}

var ethicalNotAvailable = func(r *http.Request, segment string) (*EthicalAdsAd, error) {
	return nil, nil
}

func TestFallbackCampaignAvailable(t *testing.T) {
	exp := []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
			Fallback:    true,
			Probability: 1,
		},
	}

	findSegment = emptySegment
	fetchEthicalAds = ethicalNotAvailable
	fetchBsa = bsaNotAvailable
	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return exp, nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CampaignAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
		},
	}, actual, "wrong body")
}

func TestFallbackCampaignNotAvailable(t *testing.T) {
	findSegment = emptySegment
	fetchEthicalAds = ethicalNotAvailable
	fetchBsa = bsaNotAvailable
	fetchCampaigns = campaignNotAvailable

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

func TestCampaignFail(t *testing.T) {
	findSegment = emptySegment
	fetchEthicalAds = ethicalNotAvailable
	fetchBsa = bsaNotAvailable

	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return nil, errors.New("error")
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

func TestCampaignAvailable(t *testing.T) {
	findSegment = emptySegment
	exp := []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
			Fallback:    false,
			Probability: 1,
		},
	}

	fetchBsa = bsaNotAvailable
	fetchEthicalAds = ethicalNotAvailable
	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return exp, nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CampaignAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
		},
	}, actual, "wrong body")
}

func TestCampaignAvailableByGeo(t *testing.T) {
	findSegment = emptySegment
	exp := []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
			Fallback:    false,
			Probability: 1,
			Geo:         "united states,israel,germany",
		},
	}

	getCountryByIP = func(ip string) string {
		return "united states"
	}
	fetchBsa = bsaNotAvailable
	fetchEthicalAds = ethicalNotAvailable
	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return exp, nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CampaignAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
		},
	}, actual, "wrong body")
}

func TestBsaAvailable(t *testing.T) {
	fetchEthicalAds = ethicalNotAvailable
	findSegment = emptySegment
	exp := []BsaAd{
		{
			Ad:           ad,
			Pixel:        []string{"pixel"},
			ReferralLink: "https://referral.com",
		},
	}

	fetchCampaigns = campaignNotAvailable
	fetchBsa = func(r *http.Request, propertyId string) (*BsaAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []BsaAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, exp, actual, "wrong body")
}

func TestBsaFail(t *testing.T) {
	fetchEthicalAds = ethicalNotAvailable
	findSegment = emptySegment
	exp := []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
			Fallback:    true,
			Probability: 1,
		},
	}

	fetchBsa = func(r *http.Request, propertyId string) (*BsaAd, error) {
		return nil, errors.New("error")
	}

	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return exp, nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CampaignAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
		},
	}, actual, "wrong body")
}

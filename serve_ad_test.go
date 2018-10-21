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
}

var campaignNotAvailable = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
	return nil, nil
}

var codefundNotAvailable = func(r *http.Request, propertyId string) (*CodefundAd, error) {
	return nil, nil
}

var bsaNotAvailable = func(r *http.Request) (*BsaAd, error) {
	return nil, nil
}

func TestCampaignAvailable(t *testing.T) {
	exp := []CampaignAd{
		{
			Ad:          ad,
			Placeholder: "placholder",
			Ratio:       0.5,
			Id:          "id",
		},
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
	assert.Equal(t, exp, actual, "wrong body")
}

func TestCampaignNotAvailable(t *testing.T) {
	exp := []CodefundAd{
		{
			Ad:      ad,
			Pixel:   []string{"pixel"},
			Company: "company",
		},
	}

	fetchCampaigns = campaignNotAvailable
	fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CodefundAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, exp, actual, "wrong body")
}

func TestCampaignFail(t *testing.T) {
	exp := []CodefundAd{
		{
			Ad:      ad,
			Pixel:   []string{"pixel"},
			Company: "company",
		},
	}

	fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
		return nil, errors.New("error")
	}

	fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CodefundAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, exp, actual, "wrong body")
}

func TestCodefundNotAvailable(t *testing.T) {
	exp := []BsaAd{
		{
			Ad:              ad,
			Pixel:           []string{"pixel"},
			Company:         "company",
			BackgroundColor: "#ffffff",
		},
	}

	fetchCampaigns = campaignNotAvailable
	fetchCodefund = codefundNotAvailable
	fetchBsa = func(r *http.Request) (*BsaAd, error) {
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

func TestCodefundFail(t *testing.T) {
	exp := []BsaAd{
		{
			Ad:              ad,
			Pixel:           []string{"pixel"},
			Company:         "company",
			BackgroundColor: "#ffffff",
		},
	}

	fetchCampaigns = campaignNotAvailable
	fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
		return nil, errors.New("error")
	}
	fetchBsa = func(r *http.Request) (*BsaAd, error) {
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

func TestBsaNotAvailable(t *testing.T) {
	fetchCampaigns = campaignNotAvailable
	fetchCodefund = codefundNotAvailable
	fetchBsa = bsaNotAvailable

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

func TestBsaNotFail(t *testing.T) {
	fetchCampaigns = campaignNotAvailable
	fetchCodefund = codefundNotAvailable
	fetchBsa = func(r *http.Request) (*BsaAd, error) {
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

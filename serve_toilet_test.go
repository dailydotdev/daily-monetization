package main

import (
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToiletCodefundAvailable(t *testing.T) {
	exp := []CodefundAd{
		{
			Ad:      ad,
			Pixel:   []string{"pixel"},
			Company: "company",
		},
	}

	fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createRouter()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []CodefundAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, exp, actual, "wrong body")
}

func TestToiletCodefundFail(t *testing.T) {
	exp := []BsaAd{
		{
			Ad:              ad,
			Pixel:           []string{"pixel"},
			Company:         "company",
			BackgroundColor: "#ffffff",
		},
	}

	fetchCodefund = func(r *http.Request, propertyId string) (*CodefundAd, error) {
		return nil, errors.New("error")
	}
	fetchBsa = func(r *http.Request) (*BsaAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createRouter()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []BsaAd
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, exp, actual, "wrong body")
}

func TestToiletBsaNotAvailable(t *testing.T) {
	fetchCodefund = codefundNotAvailable
	fetchBsa = bsaNotAvailable

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createRouter()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

func TestToiletBsaNotFail(t *testing.T) {
	fetchCodefund = codefundNotAvailable
	fetchBsa = func(r *http.Request) (*BsaAd, error) {
		return nil, errors.New("error")
	}

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createRouter()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	json.NewDecoder(rr.Body).Decode(&actual)
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToiletBsaAvailable(t *testing.T) {
	exp := []BsaAd{
		{
			Ad:           ad,
			Pixel:        []string{"pixel"},
			ReferralLink: "https://referral.com",
		},
	}

	fetchBsa = func(r *http.Request, propertyId string) (*BsaAd, error) {
		return &exp[0], nil
	}

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []BsaAd
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&actual))
	assert.Equal(t, exp, actual, "wrong body")
}

func TestToiletBsaNotAvailable(t *testing.T) {
	fetchBsa = bsaNotAvailable

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&actual))
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

func TestToiletBsaNotFail(t *testing.T) {
	fetchBsa = func(r *http.Request, propertyId string) (*BsaAd, error) {
		return nil, errors.New("error")
	}

	req, err := http.NewRequest("GET", "/a/toilet", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	router := createApp()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "wrong status code")

	var actual []interface{}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&actual))
	assert.Equal(t, []interface{}{}, actual, "wrong body")
}

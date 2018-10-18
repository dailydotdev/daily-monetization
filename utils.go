package main

import (
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var httpClient *http.Client

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getJson(req *http.Request, target interface{}) error {
	r, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if r.StatusCode == http.StatusOK {
		defer r.Body.Close()

		return json.NewDecoder(r.Body).Decode(target)
	} else {
		return errors.New(strconv.Itoa(r.StatusCode))
	}
}

func getJsonHystrix(breakerName string, req *http.Request, target interface{}) error {
	return hystrix.Do(breakerName,
		func() error {
			return getJson(req, target)
		}, nil)
}

// Regexp definitions
var keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)

func MarshalJSON(v interface{}) ([]byte, error) {
	marshalled, err := json.Marshal(v)

	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			// Empty keys are valid JSON, only lowercase if we do not have an
			// empty key.
			if len(match) > 2 {
				// Decode first rune after the double quotes
				r, width := utf8.DecodeRune(match[1:])
				r = unicode.ToLower(r)
				utf8.EncodeRune(match[1:width+1], r)
			}
			return match
		},
	)

	return converted, err
}

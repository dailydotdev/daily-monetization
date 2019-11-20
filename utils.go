package main

import (
	"encoding/json"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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

func getJsonHystrix(breakerName string, req *http.Request, target interface{}, ignoreNotFound bool) error {
	return hystrix.Do(breakerName,
		func() error {
			err := getJson(req, target)
			if ignoreNotFound && err != nil && err.Error() == "404" {
				return nil
			}
			return err
		}, nil)
}

// Regexp definitions
var keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)

func marshalJSON(v interface{}) ([]byte, error) {
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

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func getIpAddress(r *http.Request) string {
	ip := r.Header.Get("x-forwarded-for")
	if len(ip) == 0 {
		return r.RemoteAddr
	}
	return strings.Split(ip, ",")[0]
}

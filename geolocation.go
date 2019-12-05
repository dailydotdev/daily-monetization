package main

import "github.com/ip2location/ip2location-go"

func openGeolocationDatabase() {
	ip2location.Open("./ip2location/IP2LOCATION-LITE-DB1.BIN")
}

func closeGeolocationDatabase() {
	ip2location.Close()
}

var getCountryByIP = func(ip string) string {
	return ip2location.Get_all(ip).Country_short
}

package mcservice

import (
	"fmt"
	"io"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/httpx"
	json "github.com/goccy/go-json"
)

// maxGeoResponseBytes caps the geolocation response we are willing
// to decode.
const maxGeoResponseBytes = 1 << 20

type SunTimes struct {
	Sunrise string `json:"sunrise"`
	Sunset  string `json:"sunset"`
	IP      string `json:"ip"`
}

func GetSunTimes() SunTimes {
	// Geolocation goes over HTTPS: the previous plain-HTTP lookup
	// exposed the user's IP/coarse location unencrypted and trusted
	// an unauthenticated response.
	geoClient := httpx.NewClient(10 * time.Second)
	resp, err := geoClient.Get("https://ipapi.co/json/")
	if err != nil {
		return SunTimes{Sunrise: "06:00", Sunset: "18:00"}
	}
	defer resp.Body.Close()

	// ipapi.co's schema differs from ip-api.com's: the address is in
	// "ip", coordinates in "latitude"/"longitude", and failures come
	// back as {"error": true, "reason": ...} rather than a status
	// string.
	var geo struct {
		Error     bool    `json:"error"`
		IP        string  `json:"ip"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGeoResponseBytes+1))
	if err != nil || int64(len(body)) > maxGeoResponseBytes {
		return SunTimes{Sunrise: "06:00", Sunset: "18:00"}
	}
	if err := json.Unmarshal(body, &geo); err != nil || geo.Error {
		return SunTimes{Sunrise: "06:00", Sunset: "18:00"}
	}

	apiURL := fmt.Sprintf("https://api.sunrise-sunset.org/json?lat=%f&lng=%f&formatted=0", geo.Latitude, geo.Longitude)
	sunResp, err := httpx.Get(apiURL)
	if err != nil {
		return SunTimes{Sunrise: "06:00", Sunset: "18:00", IP: geo.IP}
	}
	defer sunResp.Body.Close()

	var sunData struct {
		Results struct {
			Sunrise string `json:"sunrise"`
			Sunset  string `json:"sunset"`
		} `json:"results"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(sunResp.Body).Decode(&sunData); err != nil || sunData.Status != "OK" {
		return SunTimes{Sunrise: "06:00", Sunset: "18:00", IP: geo.IP}
	}

	parseAndConvert := func(utcStr string) string {
		t, err := time.Parse(time.RFC3339, utcStr)
		if err != nil {
			return "00:00"
		}
		localTime := t.Local()
		return localTime.Format("15:04")
	}

	return SunTimes{
		Sunrise: parseAndConvert(sunData.Results.Sunrise),
		Sunset:  parseAndConvert(sunData.Results.Sunset),
		IP:      geo.IP,
	}
}

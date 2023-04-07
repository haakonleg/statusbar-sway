package widget

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/haakonleg/statusbar-sway/util"
)

const API_URL = "https://api.met.no/weatherapi/nowcast/2.0/complete?lat=%f&lon=%f"
const BROWSER_URL = "https://www.yr.no/nb/værvarsel/daglig-tabell/%f,%f"

var weatherIcon = map[string][]rune{
	"clearsky": {'', ''}, // nf-weather
	"cloudy":   {'', ''},
	"fair":     {'', ''},
	"fog":      {'', ''},
	"rain":     {'', '', '', ''},
	"sleet":    {'', '', '', ''},
	"snow":     {'', '', '', ''},
}

type yrWeatherData struct {
	time           time.Time
	airTemperature float64
	windSpeed      float64
	symbolCode     string
}

type Weather struct {
	*Widget

	Lat float64
	Lon float64

	weatherData     *yrWeatherData
	expireTimestamp time.Time
}

func NewWeatherWidget() *Widget {
	return newWidget("weather", -1, func(widget *Widget) impl {
		return &Weather{
			Widget: widget,
			Lat:    59.91,
			Lon:    10.75,
		}
	})
}

func (w *Weather) setup() {}

func (w *Weather) close() {}

func (w *Weather) run() {
	for {
		data, err := w.fetchWeatherData()
		if err != nil {
			log.Printf("failed to fetch weather data: %s", err.Error())
		} else {
			w.weatherData = data
			w.sendUpdate()
		}

		delay := time.Duration(rand.Intn(120)+30) * time.Second
		if !w.expireTimestamp.IsZero() {
			time.Sleep(time.Until(w.expireTimestamp) + delay)
		} else {
			time.Sleep((5 * time.Minute) + delay)
		}
	}
}

func (w *Weather) update(block *block) {
	if w.weatherData != nil {
		symbolCode := w.weatherData.symbolCode

		icon := ' '
		iconIndex := 0
		hour := w.weatherData.time.Hour()
		if hour < 8 || hour > 20 {
			iconIndex = 1
		}

		if strings.Contains(symbolCode, "thunder") {
			iconIndex += 2
		}

		if strings.Contains(symbolCode, "clearsky") {
			icon = weatherIcon["clearsky"][iconIndex]
		} else if strings.Contains(symbolCode, "cloudy") {
			icon = weatherIcon["cloudy"][iconIndex]
		} else if strings.Contains(symbolCode, "fair") {
			icon = weatherIcon["fair"][iconIndex]
		} else if strings.Contains(symbolCode, "fog") {
			icon = weatherIcon["fog"][iconIndex]
		} else if strings.Contains(symbolCode, "rain") {
			icon = weatherIcon["rain"][iconIndex]
		} else if strings.Contains(symbolCode, "sleet") {
			icon = weatherIcon["sleet"][iconIndex]
		} else if strings.Contains(symbolCode, "snow") {
			icon = weatherIcon["snow"][iconIndex]
		}

		block.FullText = fmt.Sprintf("%c  %.1f°C", icon, w.weatherData.airTemperature)
	} else {
		block.FullText = ""
	}
}

// opens weather in browser
func (w *Weather) onClick(x int, y int, btn int) {
	if btn == 1 {
		util.OpenBrowser(fmt.Sprintf(BROWSER_URL, w.Lat, w.Lon))
	}
}

func (w *Weather) fetchWeatherData() (*yrWeatherData, error) {
	log.Println("fetching new weather data")
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf(API_URL, w.Lat, w.Lon), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "gopher")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 304 {
		// not modified
		return w.weatherData, nil
	} else if res.StatusCode == 200 {
		reader := res.Body

		encoding := res.Header.Get("Content-Encoding")
		if encoding == "gzip" {
			if gzReader, err := gzip.NewReader(res.Body); err != nil {
				return nil, err
			} else {
				defer gzReader.Close()
				reader = gzReader
			}
		}

		expires := res.Header.Get("Expires")
		if expires != "" {
			if expireTimestamp, err := http.ParseTime(expires); err != nil {
				return nil, fmt.Errorf("error parsing expires header: %s", err.Error())
			} else {
				w.expireTimestamp = expireTimestamp
			}
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		weatherData, err := parseWeatherData(body)
		if err != nil {
			return nil, err
		}

		log.Printf("got weather data:\n%+v", weatherData)

		return weatherData, nil
	} else {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
}

func parseWeatherData(input []byte) (*yrWeatherData, error) {
	node, err := util.NewJsonNode(input)
	if err != nil {
		return nil, err
	}

	timeseries := node.Get("properties").Get("timeseries").Index(0)
	timeStr := timeseries.Get("time").String()
	time, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %s", timeStr)
	}

	data := timeseries.Get("data")
	details := data.Get("instant").Get("details")
	summary := data.Get("next_1_hours").Get("summary")

	return &yrWeatherData{
		time:           time,
		airTemperature: details.Get("air_temperature").Number(),
		windSpeed:      details.Get("wind_speed").Number(),
		symbolCode:     summary.Get("symbol_code").String(),
	}, nil
}

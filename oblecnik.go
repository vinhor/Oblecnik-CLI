package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/fatih/color"
)

const (
	lat = 50.080152
	lon = 14.404755
	alt = 190
)

type WeatherData struct {
	Time string `json:"time"`
	Data struct {
		Instant struct {
			Details struct {
				AirPressureAtSeaLevel float64 `json:"air_pressure_at_sea_level"`
				AirTemperature        float64 `json:"air_temperature"`
				CloudAreaFraction     float64 `json:"cloud_area_fraction"`
				RelativeHumidity      float64 `json:"relative_humidity"`
				WindFromDirection     float64 `json:"wind_from_direction"`
				WindSpeed             float64 `json:"wind_speed"`
			} `json:"details"`
		} `json:"instant"`
		Next12Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
			} `json:"details"`
		} `json:"next_12_hours"`
		Next1Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				PrecipitationAmount float64 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_1_hours"`
		Next6Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				PrecipitationAmount float64 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_6_hours"`
	} `json:"data"`
}

type ForecastData struct {
	Type     string `json:"type"`
	Geometry struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Meta struct {
			UpdatedAt string `json:"updated_at"`
			Units     struct {
				AirPressureAtSeaLevel string `json:"air_pressure_at_sea_level"`
				AirTemperature        string `json:"air_temperature"`
				CloudAreaFraction     string `json:"cloud_area_fraction"`
				PrecipitationAmount   string `json:"precipitation_amount"`
				RelativeHumidity      string `json:"relative_humidity"`
				WindFromDirection     string `json:"wind_from_direction"`
				WindSpeed             string `json:"wind_speed"`
			} `json:"units"`
		} `json:"meta"`
		Timeseries []WeatherData `json:"timeseries"`
	} `json:"properties"`
}

type WeatherSummary struct {
	Temps        []float64 // [0] = morning, [1] = noon, [2] = afternoon
	WindSpeed    float64
	RainingIndex int // 0 = sunny, 1 = cloudy, 2 = drizzle, 3 = rain
	WindIndex    int // 0 = < 8 m/s, 1 = 8-12 m/s, 2 =  > 12 m/s
}

type ClothingSummary struct {
	Hoodie        bool
	JacketIndex   int // 0 = no jacket, 1 = standard jacket, 2 = winter jacket
	TrousersIndex int // 0 = shorts, 1 = regular trousers, 2 = warm trousers
}

func main() {

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.met.no/weatherapi/locationforecast/2.0/compact?lat=%f&lon=%f&altitude=%d", lat, lon, alt), nil)
	if err != nil {
		panic("Error creating request")
	}
	req.Header.Set("User-Agent", "Oblecnik/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		panic("Error getting weather data; expected 200, got " + resp.Status)
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Error reading weather data")
	}

	var weatherData ForecastData

	err = json.Unmarshal(bytes, &weatherData)
	if err != nil {
		fmt.Printf("Response Body: %s\n", string(bytes))
		fmt.Println(err)
		panic("Error unmarshalling weather data")
	}

	weatherData.Properties.Timeseries = slices.DeleteFunc(weatherData.Properties.Timeseries, func(data WeatherData) bool {
		tGMT, _ := time.Parse(time.RFC3339, data.Time)
		t := tGMT.In(time.Now().Location())
		hour := t.Hour()
		date := t.Format("2006-01-02")
		var compareDate string
		if time.Now().Hour() > 6 {
			compareDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		} else {
			compareDate = time.Now().Format("2006-01-02")
		}
		if date != compareDate {
			return true
		}
		return hour != 7 && hour != 12 && hour != 15
	})

	var weatherSummary WeatherSummary
	weatherSummary.Temps = make([]float64, 3)
	weatherSummary.Temps[0] = weatherData.Properties.Timeseries[0].Data.Instant.Details.AirTemperature
	weatherSummary.Temps[1] = weatherData.Properties.Timeseries[1].Data.Instant.Details.AirTemperature
	weatherSummary.Temps[2] = weatherData.Properties.Timeseries[2].Data.Instant.Details.AirTemperature
	weatherSummary.WindIndex = 0
	weatherSummary.WindSpeed = 0
	weatherSummary.RainingIndex = 0

	cloudy_codes := []string{
		"partlycloudy_day",
		"partlycloudy_night",
		"partlycloudy_polartwilight",
		"cloudy",
	}

	drizzle_codes := []string{
		"lightsnowshowers_day",
		"lightsnowshowers_night",
		"lightsnowshowers_polartwilight",
		"lightrainshowers_day",
		"lightrainshowers_night",
		"lightrainshowers_polartwilight",
		"lightsleet",
		"lightsleetshowers_day",
		"lightsleetshowers_night",
		"lightsleetshowers_polartwilight",
		"lightrain",
		"fog",
		"lightrainshowersandthunder_day",
		"lightrainshowersandthunder_night",
		"lightrainshowersandthunder_polartwilight",
		"lightsnowandthunder",
		"lightssleetshowersandthunder_day",
		"lightssleetshowersandthunder_night",
		"lightssleetshowersandthunder_polartwilight",
		"lightsleetandthunder",
	}

	rain_codes := []string{
		"heavyrainandthunder",
		"heavysnowandthunder",
		"rainandthunder",
		"heavysleetshowersandthunder_day",
		"heavysleetshowersandthunder_night",
		"heavysleetshowersandthunder_polartwilight",
		"heavysnow",
		"heavyrainshowers_day",
		"heavyrainshowers_night",
		"heavyrainshowers_polartwilight",
		"heavyrain",
		"heavysleetshowers_day",
		"heavysleetshowers_night",
		"heavysleetshowers_polartwilight",
		"snow",
		"heavyrainshowersandthunder_day",
		"heavyrainshowersandthunder_night",
		"heavyrainshowersandthunder_polartwilight",
		"snowshowers_day",
		"snowshowers_night",
		"snowshowers_polartwilight",
		"snowshowersandthunder_day",
		"snowshowersandthunder_night",
		"snowshowersandthunder_polartwilight",
		"heavysleetandthunder",
		"rainshowersandthunder_day",
		"rainshowersandthunder_night",
		"rainshowersandthunder_polartwilight",
		"rain",
		"rainshowers_day",
		"rainshowers_night",
		"rainshowers_polartwilight",
		"sleetandthunder",
		"sleet",
		"sleetshowersandthunder_day",
		"sleetshowersandthunder_night",
		"sleetshowersandthunder_polartwilight",
		"rainshowersandthunder_day",
		"rainshowersandthunder_night",
		"rainshowersandthunder_polartwilight",
		"snowandthunder",
		"heavysnowshowersandthunder_day",
		"heavysnowshowersandthunder_night",
		"heavysnowshowersandthunder_polartwilight",
		"heavysnowshowers_day",
		"heavysnowshowers_night",
		"heavysnowshowers_polartwilight",
	}

	minTemp := slices.Min(weatherSummary.Temps)
	maxTemp := slices.Max(weatherSummary.Temps)

	if slices.Contains(drizzle_codes, weatherData.Properties.Timeseries[0].Data.Next12Hours.Summary.SymbolCode) && weatherSummary.RainingIndex != 3 {
		weatherSummary.RainingIndex = 2
	} else if slices.Contains(rain_codes, weatherData.Properties.Timeseries[0].Data.Next12Hours.Summary.SymbolCode) {
		weatherSummary.RainingIndex = 3
	} else if slices.Contains(cloudy_codes, weatherData.Properties.Timeseries[0].Data.Next12Hours.Summary.SymbolCode) && weatherSummary.RainingIndex == 0 {
		weatherSummary.RainingIndex = 1
	}

	for _, data := range weatherData.Properties.Timeseries {
		if data.Data.Instant.Details.WindSpeed <= 12 && data.Data.Instant.Details.WindSpeed > 8 && weatherSummary.WindIndex == 0 {
			weatherSummary.WindIndex = 1
		} else if data.Data.Instant.Details.WindSpeed > 12 {
			weatherSummary.WindIndex = 2
		}

		if weatherSummary.WindSpeed < data.Data.Instant.Details.WindSpeed {
			weatherSummary.WindSpeed = data.Data.Instant.Details.WindSpeed
		}
	}

	var clothingSummary ClothingSummary
	if maxTemp < 21 || (maxTemp < 26 && (weatherSummary.RainingIndex == 2 || weatherSummary.RainingIndex == 3)) {
		clothingSummary.Hoodie = true
	} else {
		clothingSummary.Hoodie = false
	}
	if maxTemp < 10 {
		clothingSummary.JacketIndex = 2
	} else if maxTemp < 19 || (weatherSummary.RainingIndex == 3 && minTemp >= 10) {
		clothingSummary.JacketIndex = 1
	} else {
		clothingSummary.JacketIndex = 0
	}

	if weatherSummary.RainingIndex == 0 && maxTemp > 22 {
		clothingSummary.JacketIndex = 0
	}

	if weatherSummary.WindIndex >= 1 && minTemp >= 10 {
		clothingSummary.JacketIndex = 1
	}

	if maxTemp > 25 {
		clothingSummary.TrousersIndex = 0
	} else if maxTemp > 5 {
		clothingSummary.TrousersIndex = 1
	} else {
		clothingSummary.TrousersIndex = 2
	}

	boldBlue := color.New(color.FgHiBlue).Add(color.Bold).SprintFunc()
	boldRed := color.New(color.FgHiRed).Add(color.Bold).SprintFunc()
	boldCyan := color.New(color.FgHiCyan).Add(color.Bold).SprintFunc()

	var windString string
	if weatherSummary.WindSpeed > 0 {
		windString = fmt.Sprintf("vítr až %s m/s", boldCyan(weatherSummary.WindSpeed))
	} else {
		windString = boldCyan("bezvětří")
	}

	minTempStr := fmt.Sprintf("%.1f", slices.Min(weatherSummary.Temps))
	maxTempStr := fmt.Sprintf("%.1f", slices.Max(weatherSummary.Temps))

	fmt.Printf("Teploty od %s ˚C do %s ˚C, %s, ", boldBlue(minTempStr), boldRed(maxTempStr), windString)

	switch weatherSummary.RainingIndex {
	case 0:
		color.HiYellow("slunečno")
	case 1:
		color.HiWhite("zataženo")
	case 2:
		color.HiCyan("jemný dešť")
	case 3:
		color.HiBlue("silný dešť")
	}

	inverted := color.New(color.FgBlack, color.BgWhite).SprintFunc()
	hiRed := color.New(color.FgHiRed).SprintFunc()
	hiGreen := color.New(color.FgHiGreen).SprintFunc()
	hiCyan := color.New(color.FgHiCyan).SprintFunc()
	hiBlue := color.New(color.FgHiBlue).SprintFunc()
	hiYellow := color.New(color.FgHiYellow).SprintFunc()
	hiWhite := color.New(color.FgHiWhite).SprintFunc()
	hiMagenta := color.New(color.FgHiMagenta).SprintFunc()

	fmt.Printf("\n%v\n", inverted("Oblečení:"))
	var clothesString string
	if clothingSummary.Hoodie {
		clothesString = hiRed("Mikina\n")
	} else {
		clothesString = hiGreen("Tričko\n")
	}

	switch clothingSummary.JacketIndex {
	case 1:
		clothesString += hiCyan("Tenká bunda\n")
	case 2:
		clothesString += hiBlue("Péřová bunda\n")
	}

	switch clothingSummary.TrousersIndex {
	case 0:
		clothesString += hiYellow("Kraťasy\n")
	case 1:
		clothesString += hiWhite("Kalhoty\n")
	case 2:
		clothesString += hiMagenta("Zateplené kalhoty\n")
	}

	fmt.Println(clothesString)
}

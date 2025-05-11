package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

var lat float64
var lon float64
var alt int

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
	Temps        []float64 // [0] = ráno, [1] = poledne, [2] = odpoledne
	WindSpeed    float64
	RainingIndex int // 0 = slunečno, 1 = zataženo, 2 = jemný déšť, 3 = silný déšť
	WindIndex    int // 0 = < 8 m/s, 1 = 8-12 m/s, 2 = > 12 m/s
}

type ClothingSummary struct {
	Hoodie        bool
	JacketIndex   int // 0 = bez bundy, 1 = standardní bunda, 2 = zimní bunda
	TrousersIndex int // 0 = šortky, 1 = kalhoty, 2 = teplé kalhoty
}

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error retrieving config dir")
		panic(err)
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(fmt.Sprintf("%s/Oblecnik", configDir))
	viper.SetDefault("altitude", -500)

	if len(os.Args) != 1 {
		switch os.Args[1] {
		case "set":
			setConfig()
		case "get":
			getHelp()
		}
		os.Exit(0)
	}

	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	lat = viper.GetFloat64("latitude")
	lon = viper.GetFloat64("longitude")
	alt = viper.GetInt("altitude")

	// TODO: start
	var req *http.Request
	if alt == -500 {
		req, err = http.NewRequest("GET", fmt.Sprintf("https://api.met.no/weatherapi/locationforecast/2.0/compact?lat=%f&lon=%f", lat, lon), nil)
	} else {
		req, err = http.NewRequest("GET", fmt.Sprintf("https://api.met.no/weatherapi/locationforecast/2.0/compact?lat=%f&lon=%f&altitude=%d", lat, lon, alt), nil)
	}
	if err != nil {
		panic("Chyba při vytváření HTTP požadavku")
	}
	req.Header.Set("User-Agent", "Oblecnik/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		panic("Chyba při získávání dat; očekáváno 200, ale odpověď serveru byla " + resp.Status)
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Chyba při čtení dat")
	}

	var weatherData ForecastData

	err = json.Unmarshal(bytes, &weatherData)
	if err != nil {
		fmt.Printf("Response Body: %s\n", string(bytes))
		fmt.Println(err)
		panic("Chyba při zpracování dat")
	}

	weatherData.Properties.Timeseries = slices.DeleteFunc(weatherData.Properties.Timeseries, func(data WeatherData) bool {
		tGMT, _ := time.Parse(time.RFC3339, data.Time)
		t := tGMT.In(time.Now().Location())
		hour := t.Hour()
		date := t.Format("2006-01-02")
		var compareDate string
		if time.Now().Hour() > 7 {
			compareDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		} else {
			compareDate = time.Now().Format("2006-01-02")
		}
		if date != compareDate {
			return true
		}
		return hour != 7 && hour != 12 && hour != 15
	})

	// TODO: end - make into a function

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

	clothingSummary := decideClothes(weatherSummary)

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

	date, err := time.Parse("2006-01-02T15:04:05Z", weatherData.Properties.Timeseries[0].Time)
	if err != nil {
		panic("Chyba při převádění času")
	}

	fmt.Println("Data pro", date.Format("02. 01. 2006"), "\n ")

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

func decideClothes(weatherSummary WeatherSummary) ClothingSummary {
	var clothingSummary ClothingSummary
	minTemp := slices.Min(weatherSummary.Temps)
	maxTemp := slices.Max(weatherSummary.Temps)
	if maxTemp < 21 || (maxTemp < 26 && (weatherSummary.RainingIndex == 2 || weatherSummary.RainingIndex == 3)) { // mikina pokud teplta pod 21 °C, NEBO pokud prší a teplota je pod 26 °C
		clothingSummary.Hoodie = true
	} else {
		clothingSummary.Hoodie = false
	}
	if maxTemp < 10 { // zimní bunda při teplotě pod 10 °C, při teplotě mezi 10 a 19 °C tenká bunda (nebo pokud prší a teplota je 10 °C nebo vyšší), jinak bez bundy
		clothingSummary.JacketIndex = 2
	} else if maxTemp < 15 || (weatherSummary.RainingIndex == 3 && minTemp >= 10) {
		clothingSummary.JacketIndex = 1
	} else {
		clothingSummary.JacketIndex = 0
	}

	if weatherSummary.WindIndex >= 1 && minTemp >= 10 { // bunda, pokud je silný vítr
		clothingSummary.JacketIndex = 1
	}

	if maxTemp > 25 { // při teplotě nad 25 °C šortky, nad 5 °C kalhoty, jinak zimní kalhoty
		clothingSummary.TrousersIndex = 0
	} else if maxTemp > 5 {
		clothingSummary.TrousersIndex = 1
	} else {
		clothingSummary.TrousersIndex = 2
	}
	return clothingSummary
}

func setConfig() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error retrieving config dir")
		panic(err)
	}
	lat, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		panic(err)
	}
	viper.Set("latitude", lat)

	lon, err = strconv.ParseFloat(os.Args[3], 64)
	if err != nil {
		panic(err)
	}
	viper.Set("longitude", lon)

	if len(os.Args) == 5 {
		alt, err = strconv.Atoi(os.Args[4])
		viper.Set("altitude", alt)
		if err != nil {
			panic(err)
		}
	}

	err = os.WriteFile(fmt.Sprintf("%s/Oblecnik/config.yaml", configDir), nil, 0755)
	if err != nil {
		panic(err)
	}
	err = viper.WriteConfig()
	if err != nil {
		fmt.Println("Problém při ukládání nastavení")
		panic(err)
	}
}

func getHelp() {
	fmt.Println("Oblečník - program pro výběr oblečení na základě teploty")
	fmt.Println("Použití: oblecnik")
	fmt.Println("Nastavení lokace: oblecnik set <zeměpisná šířka> <zeměpisná délka> [nadmořská výška]")
}

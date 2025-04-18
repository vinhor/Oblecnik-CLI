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

	"github.com/joho/godotenv"
)

type WeatherData struct {
	Time int `json:"dt"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		SeaLevel  int     `json:"sea_level"`
		GrndLevel int     `json:"grnd_level"`
		Humidity  int     `json:"humidity"`
		TempKf    float64 `json:"temp_kf"`
	} `json:"main"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Visibility int     `json:"visibility"`
	Pop        float64 `json:"pop"`
	Rain       struct {
		ThreeHours float64 `json:"3h"`
	} `json:"rain,omitempty"`
	Sys struct {
		Pod string `json:"pod"`
	} `json:"sys"`
	TimeText string `json:"dt_txt"`
}

type ForecastData struct {
	Code    string        `json:"cod"`
	Message int           `json:"message"`
	Cnt     int           `json:"cnt"`
	Data    []WeatherData `json:"list"`
	City    struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Coord struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"coord"`
		Country    string `json:"country"`
		Population int    `json:"population"`
		Timezone   int    `json:"timezone"`
		Sunrise    int    `json:"sunrise"`
		Sunset     int    `json:"sunset"`
	} `json:"city"`
}

type WeatherSummary struct {
	Temps        []float64 // [0] = morning, [1] = noon, [2] = afternoon
	WindSpeed    float64
	RainingIndex int // 0 = no rain, 1 = drizzle, 2 = rain
	WindIndex    int // 0 = < 8 m/s, 1 = 8-12 m/s, 2 =  > 12 m/s
}

type ClothingSummary struct {
	Hoodie        bool
	JacketIndex   int // 0 = no jacket, 1 = standard jacket, 2 = winter jacket
	TrousersIndex int // 0 = shorts, 1 = regular trousers, 2 = warm trousers
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	api_key := os.Getenv("WEATHER_KEY")

	lat_string := os.Getenv("LATITUDE")
	lon_string := os.Getenv("LONGITUDE")
	lat, err := strconv.ParseFloat(lat_string, 64)
	if err != nil {
		panic("Error parsing LATITUDE")
	}
	lon, err := strconv.ParseFloat(lon_string, 64)
	if err != nil {
		panic("Error parsing LONGITUDE")
	}

	resp, err := http.Get(fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?units=metric&cnt=8&lat=%f&lon=%f&appid=%s", lat, lon, api_key))
	if err != nil {
		fmt.Println(err)
		panic("Error getting weather data")
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic("Error reading weather data")
	}

	var weatherData ForecastData

	err = json.Unmarshal(bytes, &weatherData)
	if err != nil || weatherData.Code != "200" {
		panic("Error unmarshalling weather data")
	}

	weatherData.Data = slices.DeleteFunc(weatherData.Data, func(data WeatherData) bool {
		hour := time.Unix(int64(data.Time), 0).Hour()
		return hour < 6 || hour > 15
	})

	var weatherSummary WeatherSummary
	weatherSummary.Temps = make([]float64, 3)
	weatherSummary.Temps[0] = weatherData.Data[0].Main.Temp
	weatherSummary.Temps[1] = weatherData.Data[1].Main.Temp
	weatherSummary.Temps[2] = weatherData.Data[2].Main.Temp
	weatherSummary.WindIndex = 0
	weatherSummary.WindSpeed = 0
	weatherSummary.RainingIndex = 0

	fmt.Println(weatherData.Data)

	for _, data := range weatherData.Data {
		if slices.Contains([]int{200, 230, 231, 300, 301, 310, 311, 321, 600, 620}, data.Weather[0].ID) && weatherSummary.RainingIndex == 0 {
			weatherSummary.RainingIndex = 1
		} else if slices.Contains([]int{201, 202, 210, 211, 212, 221, 232, 302, 312, 313, 314, 321, 500, 501, 502, 503, 054, 511, 520, 521, 522, 531, 601, 602, 611, 612, 613, 615, 616, 621, 622}, data.Weather[0].ID) {
			weatherSummary.RainingIndex = 2
		} else {
			weatherSummary.RainingIndex = 0
		}

		if data.Wind.Speed <= 12 && data.Wind.Speed > 8 && weatherSummary.WindIndex == 0 {
			weatherSummary.WindIndex = 1
		} else if data.Wind.Speed > 12 {
			weatherSummary.WindIndex = 2
		}

		if weatherSummary.WindSpeed < data.Wind.Speed {
			weatherSummary.WindSpeed = data.Wind.Speed
		}
	}

	var clothingSummary ClothingSummary
	if slices.Min(weatherSummary.Temps) < 23 {
		clothingSummary.Hoodie = true
	} else {
		clothingSummary.Hoodie = false
	}
	if slices.Min(weatherSummary.Temps) < 5 {
		clothingSummary.JacketIndex = 2
	} else if slices.Min(weatherSummary.Temps) < 17 {
		clothingSummary.JacketIndex = 1
	} else {
		clothingSummary.JacketIndex = 0
	}

	if slices.Max(weatherSummary.Temps) > 25 {
		clothingSummary.TrousersIndex = 0
	} else if slices.Max(weatherSummary.Temps) > 5 {
		clothingSummary.TrousersIndex = 1
	} else {
		clothingSummary.TrousersIndex = 2
	}

	var windString string
	if weatherSummary.WindSpeed > 0 {
		windString = fmt.Sprintf("vítr až %.1f m/s", weatherSummary.WindSpeed)
	} else {
		windString = "bezvětří"
	}

	var rainString string
	switch weatherSummary.RainingIndex {
	case 0:
		rainString = "bez deště"
	case 1:
		rainString = "jemný dešť"
	case 2:
		rainString = "silný dešť"
	}

	fmt.Printf("Teploty od %.1f ˚C do %.1f ˚C, %s, %s\n", slices.Min(weatherSummary.Temps), slices.Max(weatherSummary.Temps), windString, rainString)
	fmt.Println("Oblečení:")
	var clothesString string
	if clothingSummary.Hoodie {
		clothesString = "Mikina\n"
	} else {
		clothesString = "Tričko\n"
	}

	switch clothingSummary.JacketIndex {
	case 1:
		clothesString += "Tenká bunda\n"
	case 2:
		clothesString += "Péřová bunda\n"
	}

	switch clothingSummary.TrousersIndex {
	case 0:
		clothesString += "Kraťasy\n"
	case 1:
		clothesString += "Kalhoty\n"
	case 2:
		clothesString += "Zateplené kalhoty\n"
	}

	fmt.Println(clothesString)
}

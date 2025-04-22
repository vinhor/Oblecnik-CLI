package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"testing"
	"time"
)

// Global variable for fetched data
var weatherData ForecastData

func TestMain(m *testing.M) {
	// Setup phase: Fetch weather data and handle errors gracefully
	err := fetchData()
	if err != nil {
		fmt.Println("Error fetching data:", err)
		os.Exit(1) // Exit with non-zero code to indicate failure
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// fetchData fetches the weather data from MET API
func fetchData() error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.met.no/weatherapi/locationforecast/2.0/compact?lat=%f&lon=%f&altitude=%d", lat, lon, alt), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Oblecnik/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			return fmt.Errorf("error getting weather data; expected 200, got %d", resp.StatusCode)
		}
		return fmt.Errorf("error getting weather data: %w", err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading weather data: %w", err)
	}

	err = json.Unmarshal(bytes, &weatherData)
	if err != nil {
		return fmt.Errorf("error unmarshalling weather data: %w", err)
	}

	return nil
}

func TestDataFetching(t *testing.T) {
	// Test the API call directly
	if weatherData.Properties.Timeseries == nil {
		t.Fatal("Weather data not fetched successfully")
	}
}

func TestUnmarshaling(t *testing.T) {
	// Just test if the unmarshaling process was successful
	if len(weatherData.Properties.Timeseries) == 0 {
		t.Fatal("No timeseries data available in fetched weather data")
	}
}

// TODO: combine tests, make tests into calling the function from oblecnik.go

func TestFiltering(t *testing.T) {
	// Filter the timeseries and check conditions
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

	// Validate the filtered timeseries length
	if len(weatherData.Properties.Timeseries) != 3 {
		t.Error("Expected 3 timeseries, got ", len(weatherData.Properties.Timeseries))
	}

	// Validate timeseries dates and hours
	dateTime, err := time.Parse(time.RFC3339, weatherData.Properties.Timeseries[0].Time)
	if err != nil {
		t.Errorf("Error parsing time: %v", err)
	}
	date := dateTime.Format("2006-01-02")
	if dateTime.In(time.Now().Location()).Hour() != 7 {
		t.Errorf("Expected time to be 6, got %d", dateTime.Hour())
	}

	dateTime, err = time.Parse(time.RFC3339, weatherData.Properties.Timeseries[1].Time)
	if err != nil {
		t.Errorf("Error parsing time: %v", err)
	}
	if dateTime.In(time.Now().Location()).Hour() != 12 {
		t.Errorf("Expected time to be 12, got %d", dateTime.Hour())
	}
	if dateTime.Format("2006-01-02") != date {
		t.Errorf("Expected date to be %s, got %s", date, dateTime.Format("2006-01-02"))
	}

	dateTime, err = time.Parse(time.RFC3339, weatherData.Properties.Timeseries[2].Time)
	if err != nil {
		t.Errorf("Error parsing time: %v", err)
	}
	if dateTime.In(time.Now().Location()).Hour() != 15 {
		t.Errorf("Expected time to be 15, got %d", dateTime.Hour())
	}
	if dateTime.Format("2006-01-02") != date {
		t.Errorf("Expected date to be %s, got %s", date, dateTime.Format("2006-01-02"))
	}
}

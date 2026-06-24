package model

type WeatherData struct {
	Location struct {
		Name      string `json:"name"`
		Region    string `json:"region"`
		Country   string `json:"country"`
		Localtime string `json:"localtime"`
	} `json:"location"`
	Current struct {
		TempC      float64 `json:"temp_c"`
		FeelslikeC float64 `json:"feelslike_c"`
		WindKph    float64 `json:"wind_kph"`
		WindDir    string  `json:"wind_dir"`
		Humidity   int     `json:"humidity"`
		PrecipMM   float64 `json:"precip_mm"`
		Condition  struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
	Forecast struct {
		Forecastday []struct {
			Date string `json:"date"`
			Day  struct {
				MaxtempC      float64 `json:"maxtemp_c"`
				MintempC      float64 `json:"mintemp_c"`
				MaxwindKph    float64 `json:"maxwind_kph"`
				TotalprecipMM float64 `json:"totalprecip_mm"`
				Condition     struct {
					Text string `json:"text"`
				} `json:"condition"`
			} `json:"day"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

package openmeteo

type HourlyResponse struct {
	Time          []string  `json:"time"`
	Temperature2m []float64 `json:"temperature_2m"`
	WindSpeed10m  []float64 `json:"wind_speed_10m"`
	WindGusts10m  []float64 `json:"wind_gusts_10m"`
	Precipitation []float64 `json:"precipitation"`
	PressureMsl   []float64 `json:"pressure_msl"`
	RelativeHumidity2m []float64 `json:"relative_humidity_2m"`
	CloudCover    []float64 `json:"cloud_cover"`
	WeatherCode   []int     `json:"weather_code"`
}

type DailyResponse struct {
	Time                   []string  `json:"time"`
	Temperature2mMax       []float64 `json:"temperature_2m_max"`
	Temperature2mMin       []float64 `json:"temperature_2m_min"`
	PrecipitationSum       []float64 `json:"precipitation_sum"`
	WindSpeed10mMax        []float64 `json:"wind_speed_10m_max"`
	WindGusts10mMax        []float64 `json:"wind_gusts_10m_max"`
}

type ForecastResponse struct {
	Latitude       float64        `json:"latitude"`
	Longitude      float64        `json:"longitude"`
	Elevation      float64        `json:"elevation"`
	Hourly         HourlyResponse `json:"hourly"`
}

type StationRecord struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Elevation   float64 `json:"elevation"`
}

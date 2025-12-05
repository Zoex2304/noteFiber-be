package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds API keys
type Config struct {
	GeoapifyKey    string
	BinderbyteKey  string
	Port           string
	DebugMode      bool
}

// Response structures
type CityResponse struct {
	Country string       `json:"country"`
	Cities  []CityOption `json:"cities"`
}

type CityOption struct {
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type StateResponse struct {
	City   string        `json:"city"`
	States []StateOption `json:"states"`
}

type StateOption struct {
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	Province string `json:"province,omitempty"`
}

type ZipCodeResponse struct {
	City     string          `json:"city"`
	State    string          `json:"state"`
	ZipCodes []ZipCodeOption `json:"zipcodes"`
}

type ZipCodeOption struct {
	Code    string `json:"code"`
	Area    string `json:"area,omitempty"`
	Country string `json:"country"`
}

// BinderByte API structures
type BinderbyteResponse struct {
	Code     string      `json:"code"`
	Messages string      `json:"messages"`
	Value    interface{} `json:"value"`
}

type BinderbyteProvince struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type BinderbyteCity struct {
	ID         string `json:"id"`
	IDProvinsi string `json:"id_provinsi"`
	Name       string `json:"name"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := Config{
		GeoapifyKey:   os.Getenv("GEOAPIFY_API_KEY"),
		BinderbyteKey: getEnv("BINDERBYTE_API_KEY", "free"),
		Port:          getEnv("PORT", "8080"),
		DebugMode:     getEnv("DEBUG_MODE", "false") == "true",
	}

	if config.GeoapifyKey == "" {
		log.Fatal("GEOAPIFY_API_KEY is required")
	}

	log.Printf("Server starting on port %s", config.Port)
	
	http.HandleFunc("/api/cities", corsMiddleware(getCitiesHandler(config)))
	http.HandleFunc("/api/states", corsMiddleware(getStatesHandler(config)))
	http.HandleFunc("/api/zipcodes", corsMiddleware(getZipCodesHandler(config)))
	http.HandleFunc("/api/detect-country", corsMiddleware(detectCountryHandler()))
	http.HandleFunc("/health", healthHandler)

	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func detectCountryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"country":      "ID",
			"country_name": "Indonesia",
		})
	}
}

func getCitiesHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		country := r.URL.Query().Get("country")
		query := r.URL.Query().Get("query")

		if country == "" {
			country = "ID"
		}

		if query == "" {
			http.Error(w, `{"error": "query parameter is required"}`, http.StatusBadRequest)
			return
		}

		var response CityResponse
		var err error

		if country == "ID" {
			response, err = getCitiesIndonesia(config.BinderbyteKey, query, config.DebugMode)
		} else {
			response, err = getCitiesInternational(config.GeoapifyKey, country, query, config.DebugMode)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

func getStatesHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		country := r.URL.Query().Get("country")
		city := r.URL.Query().Get("city")

		if country == "" || city == "" {
			http.Error(w, `{"error": "country and city parameters are required"}`, http.StatusBadRequest)
			return
		}

		var response StateResponse
		var err error

		if country == "ID" {
			response, err = getStatesIndonesia(config.BinderbyteKey, city, config.DebugMode)
		} else {
			response, err = getStatesInternational(config.GeoapifyKey, country, city, config.DebugMode)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

func getZipCodesHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		country := r.URL.Query().Get("country")
		city := r.URL.Query().Get("city")
		state := r.URL.Query().Get("state")

		if country == "" || city == "" {
			http.Error(w, `{"error": "country and city parameters are required"}`, http.StatusBadRequest)
			return
		}

		var response ZipCodeResponse
		var err error

		if country == "ID" {
			response, err = getZipCodesIndonesia(city, state)
		} else {
			response, err = getZipCodesInternational(config.GeoapifyKey, country, city, state, config.DebugMode)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

// Indonesia API calls
func getCitiesIndonesia(apiKey, query string, debug bool) (CityResponse, error) {
	if debug {
		log.Printf("Getting cities for query: %s", query)
	}
	
	// Get all provinces
	provincesAPI := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", apiKey)
	
	provResp, err := http.Get(provincesAPI)
	if err != nil {
		return CityResponse{}, err
	}
	defer provResp.Body.Close()

	provBody, err := io.ReadAll(provResp.Body)
	if err != nil {
		return CityResponse{}, err
	}

	var provResult BinderbyteResponse
	if err := json.Unmarshal(provBody, &provResult); err != nil {
		return CityResponse{}, err
	}

	if provResult.Code != "200" {
		return CityResponse{}, fmt.Errorf("API error: %s", provResult.Messages)
	}

	// Parse provinces
	var provinces []BinderbyteProvince
	provValueBytes, err := json.Marshal(provResult.Value)
	if err != nil {
		return CityResponse{}, err
	}

	if err := json.Unmarshal(provValueBytes, &provinces); err != nil {
		return CityResponse{}, err
	}

	if debug {
		log.Printf("Found %d provinces", len(provinces))
	}

	// Get cities for each province
	allCities := []CityOption{}
	queryLower := strings.ToLower(query)
	
	for i, province := range provinces {
		if debug && i%5 == 0 {
			log.Printf("Processing province %d/%d", i+1, len(provinces))
		}
		
		if i > 0 {
			time.Sleep(50 * time.Millisecond)
		}
		
		citiesAPI := fmt.Sprintf("http://api.binderbyte.com/wilayah/kabupaten?api_key=%s&id_provinsi=%s", 
			apiKey, province.ID)
		
		cityResp, err := http.Get(citiesAPI)
		if err != nil {
			continue
		}
		
		cityBody, err := io.ReadAll(cityResp.Body)
		cityResp.Body.Close()
		
		if err != nil {
			continue
		}
		
		var cityResult BinderbyteResponse
		if err := json.Unmarshal(cityBody, &cityResult); err != nil {
			continue
		}
		
		if cityResult.Code != "200" {
			continue
		}
		
		// Parse cities
		var cities []BinderbyteCity
		cityValueBytes, err := json.Marshal(cityResult.Value)
		if err != nil {
			continue
		}
		
		if err := json.Unmarshal(cityValueBytes, &cities); err != nil {
			continue
		}
		
		// Filter by query
		for _, city := range cities {
			if strings.Contains(strings.ToLower(city.Name), queryLower) {
				allCities = append(allCities, CityOption{
					Name:    city.Name,
					State:   province.Name,
					Country: "Indonesia",
				})
			}
		}
	}
	
	if debug {
		log.Printf("Found %d matching cities", len(allCities))
	}

	return CityResponse{
		Country: "Indonesia",
		Cities:  allCities,
	}, nil
}

func getStatesIndonesia(apiKey, city string, debug bool) (StateResponse, error) {
	if debug {
		log.Printf("Getting states for city: %s", city)
	}
	
	apiURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", apiKey)
	
	resp, err := http.Get(apiURL)
	if err != nil {
		return StateResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StateResponse{}, err
	}

	var result BinderbyteResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return StateResponse{}, err
	}

	if result.Code != "200" {
		return StateResponse{}, fmt.Errorf("API error: %s", result.Messages)
	}

	// Convert Value to []BinderbyteProvince
	valueBytes, err := json.Marshal(result.Value)
	if err != nil {
		return StateResponse{}, err
	}

	var provinces []BinderbyteProvince
	if err := json.Unmarshal(valueBytes, &provinces); err != nil {
		return StateResponse{}, err
	}

	if debug {
		log.Printf("Found %d provinces", len(provinces))
	}

	states := []StateOption{}
	for _, province := range provinces {
		states = append(states, StateOption{
			Name:     province.Name,
			Code:     province.ID,
			Province: province.Name,
		})
	}

	return StateResponse{
		City:   city,
		States: states,
	}, nil
}

func getZipCodesIndonesia(city, state string) (ZipCodeResponse, error) {
	zipCodes := []ZipCodeOption{
		{Code: "10110", Area: "Gambir", Country: "Indonesia"},
		{Code: "10120", Area: "Tanah Abang", Country: "Indonesia"},
		{Code: "10130", Area: "Menteng", Country: "Indonesia"},
	}

	return ZipCodeResponse{
		City:     city,
		State:    state,
		ZipCodes: zipCodes,
	}, nil
}

// International API calls
func getCitiesInternational(apiKey, country, query string, debug bool) (CityResponse, error) {
	if debug {
		log.Printf("Getting international cities for country: %s, query: %s", country, query)
	}
	
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", query)
	params.Add("type", "city")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", apiKey)

	fullURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return CityResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CityResponse{}, err
	}

	var result struct {
		Features []struct {
			Properties struct {
				City      string  `json:"city"`
				State     string  `json:"state"`
				Country   string  `json:"country"`
				Lon       float64 `json:"lon"`
				Lat       float64 `json:"lat"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return CityResponse{}, err
	}

	cities := []CityOption{}
	seen := make(map[string]bool)

	for _, feature := range result.Features {
		cityName := feature.Properties.City
		if cityName != "" && !seen[cityName] {
			seen[cityName] = true
			cities = append(cities, CityOption{
				Name:      cityName,
				State:     feature.Properties.State,
				Country:   feature.Properties.Country,
				Latitude:  feature.Properties.Lat,
				Longitude: feature.Properties.Lon,
			})
		}
	}

	if debug {
		log.Printf("Found %d unique cities", len(cities))
	}

	return CityResponse{
		Country: country,
		Cities:  cities,
	}, nil
}

func getStatesInternational(apiKey, country, city string, debug bool) (StateResponse, error) {
	if debug {
		log.Printf("Getting international states for country: %s, city: %s", country, city)
	}
	
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", city)
	params.Add("type", "city")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", apiKey)

	fullURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return StateResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StateResponse{}, err
	}

	var result struct {
		Features []struct {
			Properties struct {
				State     string `json:"state"`
				StateCode string `json:"state_code"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return StateResponse{}, err
	}

	states := []StateOption{}
	seen := make(map[string]bool)

	for _, feature := range result.Features {
		stateName := feature.Properties.State
		if stateName != "" && !seen[stateName] {
			seen[stateName] = true
			states = append(states, StateOption{
				Name: stateName,
				Code: feature.Properties.StateCode,
			})
		}
	}

	if debug {
		log.Printf("Found %d unique states", len(states))
	}

	return StateResponse{
		City:   city,
		States: states,
	}, nil
}

func getZipCodesInternational(apiKey, country, city, state string, debug bool) (ZipCodeResponse, error) {
	if debug {
		log.Printf("Getting zip codes for country: %s, city: %s, state: %s", country, city, state)
	}
	
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", fmt.Sprintf("%s %s", city, state))
	params.Add("type", "postcode")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", apiKey)

	fullURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return ZipCodeResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ZipCodeResponse{}, err
	}

	var result struct {
		Features []struct {
			Properties struct {
				Postcode string `json:"postcode"`
				Country  string `json:"country"`
				District string `json:"district"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ZipCodeResponse{}, err
	}

	zipCodes := []ZipCodeOption{}
	seen := make(map[string]bool)

	for _, feature := range result.Features {
		postcode := feature.Properties.Postcode
		if postcode != "" && !seen[postcode] {
			seen[postcode] = true
			zipCodes = append(zipCodes, ZipCodeOption{
				Code:    postcode,
				Area:    feature.Properties.District,
				Country: feature.Properties.Country,
			})
		}
	}

	if debug {
		log.Printf("Found %d unique zip codes", len(zipCodes))
	}

	return ZipCodeResponse{
		City:     city,
		State:    state,
		ZipCodes: zipCodes,
	}, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
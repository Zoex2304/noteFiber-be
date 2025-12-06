// FILE: internal/service/location_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ILocationService interface {
	DetectCountry(ctx context.Context) (map[string]string, error)
	GetCities(ctx context.Context, country, query string) (*dto.CityResponse, error)
	GetStates(ctx context.Context, country, city string) (*dto.StateResponse, error)
	GetZipCodes(ctx context.Context, country, city, state string) (*dto.ZipCodeResponse, error)
}

type locationService struct {
	geoapifyKey   string
	binderbyteKey string
	cache         sync.Map // In-memory cache
}

// Cache Item Wrapper
type cachedItem struct {
	data      interface{}
	expiresAt time.Time
}

func NewLocationService(geoapifyKey, binderbyteKey string) ILocationService {
	return &locationService{
		geoapifyKey:   geoapifyKey,
		binderbyteKey: binderbyteKey,
	}
}

// --- Caching Helpers ---

func (s *locationService) getFromCache(key string) (interface{}, bool) {
	val, ok := s.cache.Load(key)
	if !ok {
		return nil, false
	}
	item := val.(cachedItem)
	if time.Now().After(item.expiresAt) {
		s.cache.Delete(key)
		return nil, false
	}
	return item.data, true
}

func (s *locationService) setCache(key string, data interface{}, duration time.Duration) {
	s.cache.Store(key, cachedItem{
		data:      data,
		expiresAt: time.Now().Add(duration),
	})
}

// --- Implementations ---

func (s *locationService) DetectCountry(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"country":      "ID",
		"country_name": "Indonesia",
	}, nil
}

func (s *locationService) GetCities(ctx context.Context, country, query string) (*dto.CityResponse, error) {
	cacheKey := fmt.Sprintf("cities:%s:%s", country, query)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.CityResponse), nil
	}

	var response *dto.CityResponse
	var err error

	if country == "ID" {
		response, err = s.getCitiesIndonesia(query)
	} else {
		response, err = s.getCitiesInternational(country, query)
	}

	if err == nil {
		s.setCache(cacheKey, response, 1*time.Hour)
	}
	return response, err
}

func (s *locationService) GetStates(ctx context.Context, country, city string) (*dto.StateResponse, error) {
	cacheKey := fmt.Sprintf("states:%s:%s", country, city)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.StateResponse), nil
	}

	var response *dto.StateResponse
	var err error

	if country == "ID" {
		response, err = s.getStatesIndonesia(city)
	} else {
		response, err = s.getStatesInternational(country, city)
	}

	if err == nil {
		s.setCache(cacheKey, response, 1*time.Hour)
	}
	return response, err
}

func (s *locationService) GetZipCodes(ctx context.Context, country, city, state string) (*dto.ZipCodeResponse, error) {
	cacheKey := fmt.Sprintf("zip:%s:%s:%s", country, city, state)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.ZipCodeResponse), nil
	}

	var response *dto.ZipCodeResponse
	var err error

	if country == "ID" {
		response, err = s.getZipCodesIndonesia(city, state)
	} else {
		response, err = s.getZipCodesInternational(country, city, state)
	}

	if err == nil {
		s.setCache(cacheKey, response, 1*time.Hour)
	}
	return response, err
}

// --- Internal Helper Methods ---

type binderbyteResponse struct {
	Code     string      `json:"code"`
	Messages string      `json:"messages"`
	Value    interface{} `json:"value"`
}
type binderbyteProvince struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type binderbyteCity struct {
	ID         string `json:"id"`
	IDProvinsi string `json:"id_provinsi"`
	Name       string `json:"name"`
}

func (s *locationService) getCitiesIndonesia(query string) (*dto.CityResponse, error) {
	// 1. Fetch all provinces
	apiURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", s.binderbyteKey)
	resp, err := http.Get(apiURL)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res binderbyteResponse
	json.Unmarshal(body, &res)

	if res.Code != "200" { return nil, fmt.Errorf("binderbyte error: %s", res.Messages) }

	bProvinces, _ := json.Marshal(res.Value)
	var provinces []binderbyteProvince
	json.Unmarshal(bProvinces, &provinces)

	allCities := []dto.CityOption{}
	queryLower := strings.ToLower(query)
	
	// Thread-safe appending with mutex
	var mu sync.Mutex

	// 2. Worker Pool Optimization
	// We use a semaphore to limit concurrent requests to avoid API bans/timeouts
	// but we iterate through ALL provinces.
	maxConcurrent := 8
	sem := make(chan struct{}, maxConcurrent) 
	var wg sync.WaitGroup

	for _, p := range provinces {
		wg.Add(1)
		go func(province binderbyteProvince) {
			defer wg.Done()
			
			// Acquire semaphore (block if full)
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			cityURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/kabupaten?api_key=%s&id_provinsi=%s", s.binderbyteKey, province.ID)
			cResp, err := http.Get(cityURL)
			if err != nil { return }
			
			cBody, _ := io.ReadAll(cResp.Body)
			cResp.Body.Close()

			var cRes binderbyteResponse
			json.Unmarshal(cBody, &cRes)
			if cRes.Code != "200" { return }

			bCities, _ := json.Marshal(cRes.Value)
			var cities []binderbyteCity
			json.Unmarshal(bCities, &cities)

			// Local filtering to reduce mutex locking time
			var matches []dto.CityOption
			for _, c := range cities {
				if strings.Contains(strings.ToLower(c.Name), queryLower) {
					matches = append(matches, dto.CityOption{
						Name: c.Name,
						State: province.Name,
						Country: "Indonesia",
					})
				}
			}

			if len(matches) > 0 {
				mu.Lock()
				allCities = append(allCities, matches...)
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait() // Wait for all provinces to be processed

	return &dto.CityResponse{Country: "Indonesia", Cities: allCities}, nil
}

func (s *locationService) getCitiesInternational(country, query string) (*dto.CityResponse, error) {
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", query)
	params.Add("type", "city")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", s.geoapifyKey)

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil { return nil, err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				City string `json:"city"`
				State string `json:"state"`
				Country string `json:"country"`
				Lon float64 `json:"lon"`
				Lat float64 `json:"lat"`
			} `json:"properties"`
		} `json:"features"`
	}
	json.Unmarshal(body, &result)

	cities := []dto.CityOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		if f.Properties.City != "" && !seen[f.Properties.City] {
			seen[f.Properties.City] = true
			cities = append(cities, dto.CityOption{
				Name: f.Properties.City,
				State: f.Properties.State,
				Country: f.Properties.Country,
				Latitude: f.Properties.Lat,
				Longitude: f.Properties.Lon,
			})
		}
	}
	return &dto.CityResponse{Country: country, Cities: cities}, nil
}

func (s *locationService) getStatesIndonesia(city string) (*dto.StateResponse, error) {
	apiURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", s.binderbyteKey)
	resp, err := http.Get(apiURL)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res binderbyteResponse
	json.Unmarshal(body, &res)

	bProvinces, _ := json.Marshal(res.Value)
	var provinces []binderbyteProvince
	json.Unmarshal(bProvinces, &provinces)

	states := []dto.StateOption{}
	for _, p := range provinces {
		states = append(states, dto.StateOption{
			Name: p.Name,
			Code: p.ID,
			Province: p.Name,
		})
	}
	return &dto.StateResponse{City: city, States: states}, nil
}

func (s *locationService) getStatesInternational(country, city string) (*dto.StateResponse, error) {
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", city)
	params.Add("type", "city")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", s.geoapifyKey)

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil { return nil, err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				State string `json:"state"`
				StateCode string `json:"state_code"`
			} `json:"properties"`
		} `json:"features"`
	}
	json.Unmarshal(body, &result)

	states := []dto.StateOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		if f.Properties.State != "" && !seen[f.Properties.State] {
			seen[f.Properties.State] = true
			states = append(states, dto.StateOption{
				Name: f.Properties.State,
				Code: f.Properties.StateCode,
			})
		}
	}
	return &dto.StateResponse{City: city, States: states}, nil
}

func (s *locationService) getZipCodesIndonesia(city, state string) (*dto.ZipCodeResponse, error) {
	// Mock implementation as per original code
	zipCodes := []dto.ZipCodeOption{
		{Code: "10110", Area: "Gambir", Country: "Indonesia"},
		{Code: "10120", Area: "Tanah Abang", Country: "Indonesia"},
		{Code: "10130", Area: "Menteng", Country: "Indonesia"},
	}
	return &dto.ZipCodeResponse{City: city, State: state, ZipCodes: zipCodes}, nil
}

func (s *locationService) getZipCodesInternational(country, city, state string) (*dto.ZipCodeResponse, error) {
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", fmt.Sprintf("%s %s", city, state))
	params.Add("type", "postcode")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", s.geoapifyKey)

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil { return nil, err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				Postcode string `json:"postcode"`
				Country string `json:"country"`
				District string `json:"district"`
			} `json:"properties"`
		} `json:"features"`
	}
	json.Unmarshal(body, &result)

	zipCodes := []dto.ZipCodeOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		if f.Properties.Postcode != "" && !seen[f.Properties.Postcode] {
			seen[f.Properties.Postcode] = true
			zipCodes = append(zipCodes, dto.ZipCodeOption{
				Code: f.Properties.Postcode,
				Area: f.Properties.District,
				Country: f.Properties.Country,
			})
		}
	}
	return &dto.ZipCodeResponse{City: city, State: state, ZipCodes: zipCodes}, nil
}
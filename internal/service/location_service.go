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
	GetCountries(ctx context.Context, query string) (*dto.CountryResponse, error)
	// UPDATE: Menambahkan parameter state
	GetCities(ctx context.Context, country, query, state string) (*dto.CityResponse, error)
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

// GetCountries: Dynamic search for countries using Geoapify
func (s *locationService) GetCountries(ctx context.Context, query string) (*dto.CountryResponse, error) {
	cacheKey := fmt.Sprintf("countries:%s", query)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.CountryResponse), nil
	}

	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	params.Add("text", query)
	params.Add("type", "country")
	params.Add("apiKey", s.geoapifyKey)
	params.Add("limit", "20") // Diperbanyak dari 10 ke 20

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Features []struct {
			Properties struct {
				Country     string `json:"country"`
				CountryCode string `json:"country_code"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	countries := []dto.CountryOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		code := strings.ToUpper(f.Properties.CountryCode)
		name := f.Properties.Country

		if code != "" && !seen[code] {
			seen[code] = true
			countries = append(countries, dto.CountryOption{
				Name: name,
				Code: code,
			})
		}
	}

	response := &dto.CountryResponse{Countries: countries}
	s.setCache(cacheKey, response, 24*time.Hour)

	return response, nil
}

// UPDATED: GetCities sekarang support `state` parameter
func (s *locationService) GetCities(ctx context.Context, country, query, state string) (*dto.CityResponse, error) {
	cacheKey := fmt.Sprintf("cities:%s:%s:%s", country, query, state)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.CityResponse), nil
	}

	var response *dto.CityResponse
	var err error

	if country == "ID" {
		response, err = s.getCitiesIndonesia(query, state)
	} else {
		response, err = s.getCitiesInternational(country, query, state)
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
	// PENTING: Untuk ZipCodes, kita menggunakan Geoapify untuk SEMUA negara (termasuk Indonesia)
	// Alasan: Binderbyte butuh ID hirarkis (Prov->Kab->Kec), sedangkan input kita string nama Kota.
	// Geoapify mampu mencari zipcode berdasarkan string "City + State" dengan akurat.
	
	cacheKey := fmt.Sprintf("zip:%s:%s:%s", country, city, state)
	if val, ok := s.getFromCache(cacheKey); ok {
		return val.(*dto.ZipCodeResponse), nil
	}

	// Langsung gunakan implementasi Geoapify (Unified Logic)
	// Kita hapus logika mock "if country == ID" yang lama
	response, err := s.getZipCodesInternational(country, city, state)

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

// UPDATED: Logika Fetch Kota Indonesia
// 1. Jika `state` (ID Provinsi) ada -> Fetch langsung by ID (Cepat & Akurat untuk dropdown)
// 2. Jika `state` kosong -> Loop semua provinsi (Search text mode - Lambat)
func (s *locationService) getCitiesIndonesia(query, stateId string) (*dto.CityResponse, error) {
	
	// MODE 1: Search by State ID (Dropdown Flow)
	if stateId != "" {
		cityURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/kabupaten?api_key=%s&id_provinsi=%s", s.binderbyteKey, stateId)
		cResp, err := http.Get(cityURL)
		if err != nil { 
			return nil, err 
		}
		defer cResp.Body.Close()
		cBody, _ := io.ReadAll(cResp.Body)
		
		var cRes binderbyteResponse
		json.Unmarshal(cBody, &cRes)
		
		if cRes.Code != "200" { 
			return nil, fmt.Errorf("API Error: %s", cRes.Messages) 
		}

		bCities, _ := json.Marshal(cRes.Value)
		var cities []binderbyteCity
		json.Unmarshal(bCities, &cities)

		result := []dto.CityOption{}
		for _, c := range cities {
			// Optional: Filter query jika user mengetik di dropdown
			if query == "" || strings.Contains(strings.ToLower(c.Name), strings.ToLower(query)) {
				result = append(result, dto.CityOption{
					Name: c.Name,
					State: "Selected State", // Kita tidak tahu nama state di sini, tapi di flow dropdown frontend sudah tau
					Country: "Indonesia",
				})
			}
		}
		return &dto.CityResponse{Country: "Indonesia", Cities: result}, nil
	}

	// MODE 2: Search Global by Text (Looping - Autocomplete Flow)
	// (Kode lama tetap dipertahankan sebagai fallback jika stateId kosong)
	
	apiURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", s.binderbyteKey)
	resp, err := http.Get(apiURL)
	if err != nil { 
		return nil, err 
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res binderbyteResponse
	json.Unmarshal(body, &res)

	if res.Code != "200" { 
		return nil, fmt.Errorf("binderbyte error: %s", res.Messages) 
	}

	bProvinces, _ := json.Marshal(res.Value)
	var provinces []binderbyteProvince
	json.Unmarshal(bProvinces, &provinces)

	allCities := []dto.CityOption{}
	queryLower := strings.ToLower(query)
	var mu sync.Mutex
	maxConcurrent := 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, p := range provinces {
		wg.Add(1)
		go func(province binderbyteProvince) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cityURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/kabupaten?api_key=%s&id_provinsi=%s", s.binderbyteKey, province.ID)
			cResp, err := http.Get(cityURL)
			if err != nil { 
				return 
			}
			cBody, _ := io.ReadAll(cResp.Body)
			cResp.Body.Close()
			var cRes binderbyteResponse
			json.Unmarshal(cBody, &cRes)
			if cRes.Code != "200" { 
				return 
			}

			bCities, _ := json.Marshal(cRes.Value)
			var cities []binderbyteCity
			json.Unmarshal(bCities, &cities)

			var matches []dto.CityOption
			for _, c := range cities {
				if strings.Contains(strings.ToLower(c.Name), queryLower) {
					matches = append(matches, dto.CityOption{
						Name:    c.Name,
						State:   province.Name,
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
	wg.Wait()
	return &dto.CityResponse{Country: "Indonesia", Cities: allCities}, nil
}

// UPDATED: Support state param for international logic too
func (s *locationService) getCitiesInternational(country, query, state string) (*dto.CityResponse, error) {
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}
	
	// Jika state ada, kita prioritaskan pencarian di state tersebut
	searchText := query
	if state != "" && query == "" {
		// Jika query kosong tapi state ada, cari cities di state itu (Geoapify mungkin butuh trik khusus, tapi ini best effort)
		// Geoapify autocomplete lebih condong ke text search, jadi kita gabung state ke text
		searchText = state 
	} else if state != "" && query != "" {
		searchText = fmt.Sprintf("%s %s", query, state)
	}

	params.Add("text", searchText)
	params.Add("type", "city")
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", s.geoapifyKey)
	params.Add("limit", "30") // Diperbanyak untuk hasil kota yang lebih banyak

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				City    string  `json:"city"`
				State   string  `json:"state"`
				Country string  `json:"country"`
				Lon     float64 `json:"lon"`
				Lat     float64 `json:"lat"`
			} `json:"properties"`
		} `json:"features"`
	}
	json.Unmarshal(body, &result)

	cities := []dto.CityOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		// Validasi tambahan: Pastikan nama kota tidak kosong
		if f.Properties.City != "" && !seen[f.Properties.City] {
			seen[f.Properties.City] = true
			cities = append(cities, dto.CityOption{
				Name:      f.Properties.City,
				State:     f.Properties.State,
				Country:   f.Properties.Country,
				Latitude:  f.Properties.Lat,
				Longitude: f.Properties.Lon,
			})
		}
	}
	return &dto.CityResponse{Country: country, Cities: cities}, nil
}

func (s *locationService) getStatesIndonesia(city string) (*dto.StateResponse, error) {
	apiURL := fmt.Sprintf("http://api.binderbyte.com/wilayah/provinsi?api_key=%s", s.binderbyteKey)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
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
			Name:     p.Name,
			Code:     p.ID,
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
	params.Add("limit", "30") // Diperbanyak untuk hasil state yang lebih banyak

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				State     string `json:"state"`
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

// Method ini sekarang dipakai untuk SEMUA negara, termasuk Indonesia
func (s *locationService) getZipCodesInternational(country, city, state string) (*dto.ZipCodeResponse, error) {
	baseURL := "https://api.geoapify.com/v1/geocode/autocomplete"
	params := url.Values{}

	// Gabungkan Kota dan Provinsi untuk pencarian yang lebih akurat
	searchText := fmt.Sprintf("%s %s", city, state)
	params.Add("text", searchText)
	params.Add("type", "postcode") // Cari spesifik postcode
	params.Add("filter", "countrycode:"+strings.ToLower(country))
	params.Add("apiKey", s.geoapifyKey)
	params.Add("limit", "50") // Diperbanyak menjadi 50 untuk hasil zip code yang lebih banyak

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Features []struct {
			Properties struct {
				Postcode string `json:"postcode"`
				Country  string `json:"country"`
				District string `json:"district"` // Kecamatan/Area
				County   string `json:"county"`   // Kabupaten/Kota
				State    string `json:"state"`
			} `json:"properties"`
		} `json:"features"`
	}
	json.Unmarshal(body, &result)

	zipCodes := []dto.ZipCodeOption{}
	seen := make(map[string]bool)

	for _, f := range result.Features {
		if f.Properties.Postcode != "" && !seen[f.Properties.Postcode] {
			seen[f.Properties.Postcode] = true

			// Ambil Area terbaik (District/Kecamatan jika ada, atau County)
			area := f.Properties.District
			if area == "" {
				area = f.Properties.County
			}

			zipCodes = append(zipCodes, dto.ZipCodeOption{
				Code:    f.Properties.Postcode,
				Area:    area,
				Country: f.Properties.Country,
			})
		}
	}
	return &dto.ZipCodeResponse{City: city, State: state, ZipCodes: zipCodes}, nil
}
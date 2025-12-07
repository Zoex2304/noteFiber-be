// FILE: internal/dto/location_dto.go
package dto

// Location API DTOs

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

// DTO Baru untuk Negara
type CountryResponse struct {
	Countries []CountryOption `json:"countries"`
}

type CountryOption struct {
	Name string `json:"name"` // Contoh: "Indonesia"
	Code string `json:"code"` // Contoh: "ID", "SG", "US"
	Flag string `json:"flag,omitempty"` // Opsional: emoji bendera atau url
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
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	fmt.Println("=================================")
	fmt.Println("  List Available Gemini Models")
	fmt.Println("=================================")
	
	var apiKey string
	fmt.Print("\nEnter your Gemini API Key: ")
	fmt.Scanln(&apiKey)
	
	if apiKey == "" {
		fmt.Println("❌ API Key cannot be empty!")
		return
	}
	
	// List models from v1beta
	fmt.Println("\n=== Available Models (v1beta) ===")
	listModels(apiKey, "v1beta")
	
	// List models from v1
	fmt.Println("\n=== Available Models (v1) ===")
	listModels(apiKey, "v1")
}

func listModels(apiKey, version string) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models?key=%s", version, apiKey)
	
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	
	body, _ := io.ReadAll(res.Body)
	
	if res.StatusCode != 200 {
		fmt.Printf("❌ Status: %d\n", res.StatusCode)
		fmt.Println("Response:", string(body))
		return
	}
	
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	models := result["models"].([]interface{})
	
	fmt.Printf("\nFound %d models:\n\n", len(models))
	
	for _, m := range models {
		model := m.(map[string]interface{})
		name := model["name"].(string)
		
		// Check supported methods
		methods := []string{}
		if supportedMethods, ok := model["supportedGenerationMethods"].([]interface{}); ok {
			for _, method := range supportedMethods {
				methods = append(methods, method.(string))
			}
		}
		
		// Filter only models that support generateContent
		hasGenerateContent := false
		for _, method := range methods {
			if method == "generateContent" {
				hasGenerateContent = true
				break
			}
		}
		
		if hasGenerateContent {
			fmt.Printf("✅ %s\n", name)
			fmt.Printf("   Methods: %v\n\n", methods)
		}
	}
}
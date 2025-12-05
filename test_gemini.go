package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func testChat(apiKey string) {
	fmt.Println("\n=== Testing Gemini Chat (gemini-2.0-flash) ===")
	
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": "Hello! Say 'Working!' if you can read this."},
				},
			},
		},
	}
	
	payloadJson, _ := json.Marshal(payload)
	
	// VERIFIED: v1beta endpoint with gemini-2.0-flash (from ListModels)
	req, _ := http.NewRequest(
		"POST",
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent",
		bytes.NewBuffer(payloadJson),
	)
	
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	res, err := client.Do(req)
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
	
	candidates := result["candidates"].([]interface{})
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	text := parts[0].(map[string]interface{})["text"].(string)
	
	fmt.Println("✅ Chat Response:", text)
}

func testEmbedding(apiKey string) {
	fmt.Println("\n=== Testing Gemini Embedding (text-embedding-004) ===")
	
	payload := map[string]interface{}{
		"model": "models/text-embedding-004",
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": "Hello world"},
			},
		},
	}
	
	payloadJson, _ := json.Marshal(payload)
	
	// OFFICIAL: v1beta endpoint with text-embedding-004
	req, _ := http.NewRequest(
		"POST",
		"https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent",
		bytes.NewBuffer(payloadJson),
	)
	
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	res, err := client.Do(req)
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
	
	embedding := result["embedding"].(map[string]interface{})
	values := embedding["values"].([]interface{})
	
	fmt.Printf("✅ Embedding Generated: %d dimensions\n", len(values))
	fmt.Printf("   First 5 values: %.4f, %.4f, %.4f, %.4f, %.4f\n", 
		values[0].(float64), values[1].(float64), values[2].(float64), 
		values[3].(float64), values[4].(float64))
}

func main() {
	fmt.Println("=================================")
	fmt.Println("  Gemini API Testing Tool")
	fmt.Println("  Based on Official Docs")
	fmt.Println("=================================")
	
	var apiKey string
	fmt.Print("\nEnter your Gemini API Key: ")
	fmt.Scanln(&apiKey)
	
	if apiKey == "" {
		fmt.Println("❌ API Key cannot be empty!")
		return
	}
	
	testChat(apiKey)
	testEmbedding(apiKey)
	
	fmt.Println("\n=================================")
	fmt.Println("  Testing Complete!")
	fmt.Println("=================================")
	fmt.Println("\nModels tested:")
	fmt.Println("  • Chat: gemini-1.5-flash")
	fmt.Println("  • Embedding: text-embedding-004")
}
// FILE: internal/pkg/logger/csv_logger.go
package logger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
)

type LogEntry struct {
	Id        string
	Timestamp string
	Level     string
	Module    string
	Message   string
	Details   string // JSON string
}

type ILogger interface {
	Debug(module, message string, details map[string]interface{})
	Info(module, message string, details map[string]interface{})
	Error(module, message string, details map[string]interface{})
	GetLogs(level string, limit, offset int) ([]LogEntry, error)
	GetLogById(id string) (*LogEntry, error)
}

type CsvLogger struct {
	filePath string
	mu       sync.Mutex
}

func NewCsvLogger(filePath string) *CsvLogger {
	// Ensure file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("FAILED TO CREATE LOG FILE: %v\n", err)
		} else {
			writer := csv.NewWriter(file)
			// Header
			writer.Write([]string{"id", "timestamp", "level", "module", "message", "details"})
			writer.Flush()
			file.Close()
		}
	}
	return &CsvLogger{filePath: filePath}
}

func (l *CsvLogger) write(level LogLevel, module, message string, details map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("FAILED TO OPEN LOG FILE: %v\n", err)
		return
	}
	defer file.Close()

	detailsJson, _ := json.Marshal(details)
	record := []string{
		uuid.New().String(),
		time.Now().Format(time.RFC3339),
		string(level),
		module,
		message,
		string(detailsJson),
	}

	writer := csv.NewWriter(file)
	writer.Write(record)
	writer.Flush()
}

func (l *CsvLogger) Debug(module, message string, details map[string]interface{}) {
	l.write(DEBUG, module, message, details)
}

func (l *CsvLogger) Info(module, message string, details map[string]interface{}) {
	l.write(INFO, module, message, details)
}

func (l *CsvLogger) Error(module, message string, details map[string]interface{}) {
	l.write(ERROR, module, message, details)
}

func (l *CsvLogger) GetLogs(level string, limit, offset int) ([]LogEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []LogEntry
	// Start from end (newest first), skip header (index 0)
	for i := len(records) - 1; i > 0; i-- {
		rec := records[i]
		if len(rec) < 6 {
			continue
		}
		
		if level != "" && rec[2] != level {
			continue
		}

		filtered = append(filtered, LogEntry{
			Id:        rec[0],
			Timestamp: rec[1],
			Level:     rec[2],
			Module:    rec[3],
			Message:   rec[4],
			Details:   rec[5],
		})
	}

	// Pagination
	start := offset
	end := offset + limit
	if start >= len(filtered) {
		return []LogEntry{}, nil
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

func (l *CsvLogger) GetLogById(id string) (*LogEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, rec := range records {
		if len(rec) >= 6 && rec[0] == id {
			return &LogEntry{
				Id:        rec[0],
				Timestamp: rec[1],
				Level:     rec[2],
				Module:    rec[3],
				Message:   rec[4],
				Details:   rec[5],
			}, nil
		}
	}
	return nil, fmt.Errorf("log not found")
}
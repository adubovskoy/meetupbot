package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// Config represents the bot configuration
type Config struct {
	BotToken        string
	AdminUsers      []string
	MandatoryFields []string
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		AdminUsers:      []string{},
		MandatoryFields: []string{},
	}

	// Try to load from .env file
	if err := loadEnvFile(".env"); err == nil {
		log.Println("Loaded .env file")
	}

	// Get configuration from environment variables
	config.BotToken = os.Getenv("BOT_TOKEN")

	if adminUsers := os.Getenv("ADMIN_USERS"); adminUsers != "" {
		config.AdminUsers = parseCommaSeparated(adminUsers)
	}

	if mandatoryFields := os.Getenv("MANDATORY_FIELDS"); mandatoryFields != "" {
		config.MandatoryFields = parseCommaSeparated(mandatoryFields)
	}

	// Validate configuration
	if config.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	// Validate mandatory fields
	validFields := map[string]bool{
		"name":  true,
		"email": true,
	}
	for _, field := range config.MandatoryFields {
		if !validFields[strings.ToLower(field)] {
			return nil, fmt.Errorf("invalid mandatory field: %s", field)
		}
	}

	return config, nil
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Remove quotes if present
			value = strings.Trim(value, `"'`)
			
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}



// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	
	return result
}

// HasMandatoryField checks if a field is mandatory
func (c *Config) HasMandatoryField(field string) bool {
	field = strings.ToLower(field)
	for _, mf := range c.MandatoryFields {
		if strings.ToLower(mf) == field {
			return true
		}
	}
	return false
}

// RequiresDialog checks if any mandatory fields require a dialog
func (c *Config) RequiresDialog() bool {
	return len(c.MandatoryFields) > 0
}
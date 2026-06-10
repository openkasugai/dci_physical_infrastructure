// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Config holds all parsed and validated environment variable configuration
type Config struct {
	LogLevel        string
	Interval        int // Parsed as integer
	P2PEnable       bool
	P2PInterval     int // Parsed as integer
	SshKey          string
	MetricsPort     int // Parsed as integer
	MetricsEndpoint string
	DbAccessURL     string
}

// EnvConfig holds raw environment variable strings for validation
type EnvConfig struct {
	LogLevel        string `validate:"required" env:"LOG_LEVEL"`
	Interval        string `validate:"required" env:"INTERVAL"`
	P2PEnable       string `validate:"required" env:"P2P_ENABLE"`
	P2PInterval     string `validate:"required" env:"P2P_INTERVAL"`
	SshKey          string `validate:"required" env:"SSH_KEY"`
	MetricsPort     string `validate:"required" env:"METRICS_PORT"`
	MetricsEndpoint string `validate:"required" env:"METRICS_ENDPOINT"`
	DbAccessURL     string `validate:"required" env:"DB_URL"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
	configError  error
)

// EnvValidator provides environment variable validation functionality
type EnvValidator struct {
	validator *validator.Validate
}

// InitializeConfig initializes the global configuration from environment variables
// This should be called once at application startup
func InitializeConfig() error {
	configOnce.Do(func() {
		validator := newEnvValidator()
		globalConfig, configError = validator.loadAndValidateConfig()
	})
	return configError
}

// GetConfig returns the global configuration
// InitializeConfig must be called before using this function
func GetConfig() *Config {
	return globalConfig
}

// ResetConfigForTesting resets the global configuration state for testing purposes
// This should only be used in test code
func ResetConfigForTesting() {
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil
}

// loadAndValidateConfig loads environment variables, validates them, and returns parsed Config
func (ev *EnvValidator) loadAndValidateConfig() (*Config, error) {
	// Load raw environment variables
	rawConfig := &EnvConfig{
		LogLevel:        os.Getenv("LOG_LEVEL"),
		Interval:        os.Getenv("INTERVAL"),
		P2PEnable:       os.Getenv("P2P_ENABLE"),
		P2PInterval:     os.Getenv("P2P_INTERVAL"),
		SshKey:          os.Getenv("SSH_KEY"),
		MetricsPort:     os.Getenv("METRICS_PORT"),
		MetricsEndpoint: os.Getenv("METRICS_ENDPOINT"),
		DbAccessURL:     os.Getenv("DB_URL"),
	}

	// Run struct validation for required fields
	if err := ev.validator.Struct(rawConfig); err != nil {
		return nil, ev.formatValidationError(err)
	}

	// Parse and validate log level
	logLevel, err := ev.parseAndValidateLogLevel(rawConfig.LogLevel)
	if err != nil {
		return nil, err
	}

	// Parse and validate interval
	interval, err := ev.parseAndValidateInterval("interval", rawConfig.Interval)
	if err != nil {
		return nil, err
	}

	// Parse and validate P2P enable
	p2pEnable, err := ev.parseAndValidateP2PEnable(rawConfig.P2PEnable)
	if err != nil {
		return nil, err
	}
	
	// Parse and validate P2P interval
	p2pInterval, err := ev.parseAndValidateInterval("p2pInterval", rawConfig.P2PInterval)
	if err != nil {
		return nil, err
	}

	// Parse and validate metrics port
	metricsPort, err := ev.parseAndValidatePort("metrics.port", rawConfig.MetricsPort)
	if err != nil {
		return nil, err
	}

	// Create the parsed config
	config := &Config{
		LogLevel:        logLevel,
		Interval:        interval,
		P2PEnable:       p2pEnable,
		P2PInterval:     p2pInterval,
		SshKey:          rawConfig.SshKey,
		MetricsPort:     metricsPort,
		MetricsEndpoint: rawConfig.MetricsEndpoint,
		DbAccessURL:     rawConfig.DbAccessURL,
	}

	return config, nil
}

// NewEnvValidator creates a new instance of EnvValidator
func newEnvValidator() *EnvValidator {
	return &EnvValidator{
		validator: validator.New(),
	}
}

// parseAndValidateLogLevel validates and parses the log level value
func (ev *EnvValidator) parseAndValidateLogLevel(level string) (string, error) {
	message := "invalid logLevel of configuration: value must be integer string and between 0 ～ 9, inclusive"

	logLevel, err := strconv.Atoi(level)
	if err != nil {
		return "0", errors.New(message)
	}

	if logLevel < 0 || logLevel > 9 {
		return "0", errors.New(message)
	}

	return level, nil
}

// parseAndValidateP2PEnable validates and parses the P2P enable value
func (ev *EnvValidator) parseAndValidateP2PEnable(value string) (bool, error) {
	lowerValue := strings.ToLower(value)
	if lowerValue == "true" {
		return true, nil
	} else if lowerValue == "false" {
		return false, nil
	} else {
		return false, errors.New("invalid P2P_ENABLE of configuration: value must be boolean")
	}
}

// parseAndValidateInterval validates and parses the interval value
func (ev *EnvValidator) parseAndValidateInterval(key string, value string) (int, error) {
	message := "invalid " + key + " of configuration: value must be integer and between 1 ～ 3600, inclusive"

	interval, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if interval < 1 || interval > 3600 {
		return 0, errors.New(message)
	}

	return interval, nil
}

// parseAndValidatePort validates and parses the port value
func (ev *EnvValidator) parseAndValidatePort(key string, value string) (int, error) {
	message := "invalid " + key + " of configuration: value must be integer  and between 0 ～ 65535,  inclusive"

	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if port < 0 || port > 65535 {
		return 0, errors.New(message)
	}

	return port, nil
}

// formatValidationError formats validator errors into a readable message
func (ev *EnvValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, validationError := range validationErrors {
			switch validationError.Tag() {
			case "required":
				envVar := ev.getEnvVarName(validationError.Field())
				errorMessages = append(errorMessages, fmt.Sprintf("%s is required", envVar))
			default:
				errorMessages = append(errorMessages, fmt.Sprintf("validation failed for field %s with tag %s", validationError.Field(), validationError.Tag()))
			}
		}
	}

	return errors.New(strings.Join(errorMessages, ", "))
}

// getEnvVarName returns the environment variable name for a struct field
func (ev *EnvValidator) getEnvVarName(fieldName string) string {
	switch fieldName {
	case "Interval":
		return "INTERVAL"
	case "P2PEnable":
		return "P2P_ENABLE"
	case "P2PInterval":
		return "P2P_INTERVAL"
	case "MetricsPort":
		return "METRICS_PORT"
	case "MetricsEndpoint":
		return "METRICS_ENDPOINT"
	case "DbAccessURL":
		return "DB_URL"
	case "SshKey":
		return "SSH_KEY"
	case "LogLevel":
		return "LOG_LEVEL"
	default:
		return fieldName
	}
}

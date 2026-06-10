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
	IpmiLogFile     string
	IpmiLogPath     string
	IpmiMaxSize     int // Parsed as integer
	IpmiMaxBackups  int // Parsed as integer
	IpmiMaxAge      int // Parsed as integer
	CdiLogFile      string
	CdiLogPath      string
	CdiMaxSize      int // Parsed as integer
	CdiMaxBackups   int // Parsed as integer
	CdiMaxAge       int // Parsed as integer
	DbAccessURL     string
}

// EnvConfig holds raw environment variable strings for validation
type EnvConfig struct {
	LogLevel       string `validate:"required" env:"LOG_LEVEL"`
	Interval       string `validate:"required" env:"INTERVAL"`
	IpmiLogFile    string `validate:"required" env:"IPMI_LOGFILE"`
	IpmiLogPath    string `validate:"required" env:"IPMI_LOGPATH"`
	IpmiMaxSize    string `validate:"required" env:"IPMI_MAXSIZE"`
	IpmiMaxBackups string `validate:"required" env:"IPMI_MAXBACKUPS"`
	IpmiMaxAge     string `validate:"required" env:"IPMI_MAXAGE"`
	CdiLogFile     string `validate:"required" env:"CDI_LOGFILE"`
	CdiLogPath     string `validate:"required" env:"CDI_LOGPATH"`
	CdiMaxSize     string `validate:"required" env:"CDI_MAXSIZE"`
	CdiMaxBackups  string `validate:"required" env:"CDI_MAXBACKUPS"`
	CdiMaxAge      string `validate:"required" env:"CDI_MAXAGE"`
	DbAccessURL    string `validate:"required" env:"DB_URL"`
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
		IpmiLogFile:     os.Getenv("IPMI_LOGFILE"),
		IpmiLogPath:     os.Getenv("IPMI_LOGPATH"),
		IpmiMaxSize:     os.Getenv("IPMI_MAXSIZE"),
		IpmiMaxBackups:  os.Getenv("IPMI_MAXBACKUPS"),
		IpmiMaxAge:      os.Getenv("IPMI_MAXAGE"),
		CdiLogFile:      os.Getenv("CDI_LOGFILE"),
		CdiLogPath:      os.Getenv("CDI_LOGPATH"),
		CdiMaxSize:      os.Getenv("CDI_MAXSIZE"),
		CdiMaxBackups:   os.Getenv("CDI_MAXBACKUPS"),
		CdiMaxAge:       os.Getenv("CDI_MAXAGE"),
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
	interval, err := ev.parseAndValidateInterval(rawConfig.Interval)
	if err != nil {
		return nil, err
	}

	// Parse and validate ipmi max size
	ipmiMaxSize, err := ev.parseAndValidateMaxSize("ipmi.maxSize", rawConfig.IpmiMaxSize)
	if err != nil {
		return nil, err
	}

	// Parse and validate ipmi max backups
	ipmiMaxBackups, err := ev.parseAndValidateMaxBackups("ipmi.maxBackups", rawConfig.IpmiMaxBackups)
	if err != nil {
		return nil, err
	}

	// Parse and validate ipmi max age
	ipmiMaxAge, err := ev.parseAndValidateMaxAge("ipmi.maxAge", rawConfig.IpmiMaxAge)
	if err != nil {
		return nil, err
	}

	// Parse and validate cdi max size
	cdiMaxSize, err := ev.parseAndValidateMaxSize("cdi.maxSize", rawConfig.CdiMaxSize)
	if err != nil {
		return nil, err
	}

	// Parse and validate cdi max backups
	cdiMaxBackups, err := ev.parseAndValidateMaxBackups("cdi.maxBackups", rawConfig.CdiMaxBackups)
	if err != nil {
		return nil, err
	}

	// Parse and validate cdi max age
	cdiMaxAge, err := ev.parseAndValidateMaxAge("cdi.maxAge", rawConfig.CdiMaxAge)
	if err != nil {
		return nil, err
	}

	// Create the parsed config
	config := &Config{
		LogLevel:        logLevel,
		Interval:        interval,
		IpmiLogFile:     rawConfig.IpmiLogFile,
		IpmiLogPath:     rawConfig.IpmiLogPath,
		IpmiMaxSize:     ipmiMaxSize,
		IpmiMaxBackups:  ipmiMaxBackups,
		IpmiMaxAge:      ipmiMaxAge,
		CdiLogFile:      rawConfig.CdiLogFile,
		CdiLogPath:      rawConfig.CdiLogPath,
		CdiMaxSize:      cdiMaxSize,
		CdiMaxBackups:   cdiMaxBackups,
		CdiMaxAge:       cdiMaxAge,
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

// parseAndValidateInterval validates and parses the interval value
func (ev *EnvValidator) parseAndValidateInterval(value string) (int, error) {
	message := "invalid interval of configuration: value must be integer and between 1 ～ 3600, inclusive"

	interval, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if interval < 1 || interval > 3600 {
		return 0, errors.New(message)
	}

	return interval, nil
}

// parseAndValidateMaxSize validates and parses the port value
func (ev *EnvValidator) parseAndValidateMaxSize(key string, value string) (int, error) {
	message := "invalid " + key + " of configuration: value must be integer and between 1 ～ 10240, inclusive"

	maxSize, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if maxSize < 1 || maxSize > 10240 {
		return 0, errors.New(message)
	}

	return maxSize, nil
}

// parseAndValidateMaxBackups validates and parses the port value
func (ev *EnvValidator) parseAndValidateMaxBackups(key string, value string) (int, error) {
	message := "invalid " + key + " of configuration: value must be integer and between 1 ～ 31, inclusive"

	maxBackups, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if maxBackups < 1 || maxBackups > 31 {
		return 0, errors.New(message)
	}

	return maxBackups, nil
}

// parseAndValidateMaxAge validates and parses the port value
func (ev *EnvValidator) parseAndValidateMaxAge(key string, value string) (int, error) {
	message := "invalid " + key + " of configuration: value must be integer and between 1 ～ 31, inclusive"

	maxAge, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if maxAge < 1 || maxAge > 31 {
		return 0, errors.New(message)
	}

	return maxAge, nil
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
	case "LogLevel":
		return "LOG_LEVEL"
	case "Interval":
		return "INTERVAL"
	case "IpmiLogFile":
		return "IPMI_LOGFILE"
	case "IpmiLogPath":
		return "IPMI_LOGPATH"
	case "IpmiMaxSize":
		return "IPMI_MAXSIZE"
	case "IpmiMaxBackups":
		return "IPMI_MAXBACKUPS"
	case "IpmiMaxAge":
		return "IPMI_MAXAGE"
	case "CdiLogFile":
		return "CDI_LOGFILE"
	case "CdiLogPath":
		return "CDI_LOGPATH"
	case "CdiMaxSize":
		return "CDI_MAXSIZE"
	case "CdiMaxBackups":
		return "CDI_MAXBACKUPS"
	case "CdiMaxAge":
		return "CDI_MAXAGE"
	case "DbAccessURL":
		return "DB_URL"
	default:
		return fieldName
	}
}

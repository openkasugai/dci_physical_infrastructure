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
	NWServerPort int // Parsed as integer
	LogLevel     string
	SSHKey       string
	TlsEnable    bool   // TLS enabled/disabled flag
	TlsCertPath  string // TLS certificate directory path
}

// EnvConfig holds raw environment variable strings for validation
type EnvConfig struct {
	NWServerPort string `validate:"required" env:"NW_SERVER_PORT"`
	LogLevel     string `validate:"required" env:"LOG_LEVEL"`
	SSHKey       string `validate:"required" env:"SSH_KEY"`
	TlsEnable    string `validate:"required" env:"TLS_ENABLE"`
	TlsCertPath  string `validate:"required" env:"TLS_CERT_PATH"`
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
// This should only be used in test code to allow re-initialization of config
func ResetConfigForTesting() {
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil
}

// loadAndValidateConfig loads environment variables, validates them, and returns parsed Config
func (ev *EnvValidator) loadAndValidateConfig() (*Config, error) {
	// Load raw environment variables
	rawConfig := &EnvConfig{
		NWServerPort: os.Getenv("NW_SERVER_PORT"),
		LogLevel:     os.Getenv("LOG_LEVEL"),
		SSHKey:       os.Getenv("SSH_KEY"),
		TlsEnable:    os.Getenv("TLS_ENABLE"),
		TlsCertPath:  os.Getenv("TLS_CERT_PATH"),
	}

	// Run struct validation for required fields
	if err := ev.validator.Struct(rawConfig); err != nil {
		return nil, ev.formatValidationError(err)
	}

	// Parse and validate NW server port
	serverPort, err := ev.parseAndValidateNWServerPort(rawConfig.NWServerPort)
	if err != nil {
		return nil, err
	}

	// Parse and validate log level
	logLevel, err := ev.parseAndValidateLogLevel(rawConfig.LogLevel)
	if err != nil {
		return nil, err
	}

	// Parse and validate TLS enable flag
	tlsEnable, err := ev.parseAndValidateTlsEnable(rawConfig.TlsEnable)
	if err != nil {
		return nil, err
	}

	// Create the parsed config
	config := &Config{
		NWServerPort: serverPort,
		LogLevel:     logLevel,
		SSHKey:       rawConfig.SSHKey,
		TlsEnable:    tlsEnable,
		TlsCertPath:  rawConfig.TlsCertPath,
	}

	return config, nil
}

// NewEnvValidator creates a new instance of EnvValidator
func newEnvValidator() *EnvValidator {
	return &EnvValidator{
		validator: validator.New(),
	}
}

// parseAndValidateNWServerPort validates and parses the NW server port value
func (ev *EnvValidator) parseAndValidateNWServerPort(port string) (int, error) {
	message := "invalid serverPort of configuration: value must be integer and between 1 ～ 65535, inclusive"

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return 0, errors.New(message)
	}

	if portNum < 1 || portNum > 65535 {
		return 0, errors.New(message)
	}

	return portNum, nil
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

// parseAndValidateTlsEnable validates and parses the TLS enable flag
func (ev *EnvValidator) parseAndValidateTlsEnable(value string) (bool, error) {
	message := "invalid tlsEnable of configuration: value must be boolean (true or false)"

	tlsEnable, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New(message)
	}

	return tlsEnable, nil
}

// Legacy methods for backward compatibility
// validateNWServerPort validates the NW server port value
func (ev *EnvValidator) validateNWServerPort(port string) error {
	_, err := ev.parseAndValidateNWServerPort(port)
	return err
}

// validateLogLevel validates the log level value
func (ev *EnvValidator) validateLogLevel(level string) error {
	_, err := ev.parseAndValidateLogLevel(level)
	return err
}

// ValidateEnvironment validates all required environment variables (legacy method)
// Deprecated: Use InitializeConfig() instead for better performance and type safety
func (ev *EnvValidator) validateEnvironment() error {
	config := &EnvConfig{
		NWServerPort: os.Getenv("NW_SERVER_PORT"),
		LogLevel:     os.Getenv("LOG_LEVEL"),
		SSHKey:       os.Getenv("SSH_KEY"),
		TlsEnable:    os.Getenv("TLS_ENABLE"),
		TlsCertPath:  os.Getenv("TLS_CERT_PATH"),
	}

	// Run struct validation for required fields
	if err := ev.validator.Struct(config); err != nil {
		return ev.formatValidationError(err)
	}

	// Custom validation logic
	if err := ev.validateNWServerPort(config.NWServerPort); err != nil {
		return err
	}
	if err := ev.validateLogLevel(config.LogLevel); err != nil {
		return err
	}

	return nil
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
	case "NWServerPort":
		return "NW_SERVER_PORT"
	case "LogLevel":
		return "LOG_LEVEL"
	case "SSHKey":
		return "SSH_KEY"
	case "TlsEnable":
		return "TLS_ENABLE"
	case "TlsCertPath":
		return "TLS_CERT_PATH"
	default:
		return fieldName
	}
}

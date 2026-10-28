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
	LogLevel                 string
	ServerPort               int // Parsed as integer
	VmHostDisk               int // Parsed as integer
	LxdListenPort            int // Parsed as integer
	SshKey                   string
	TlsEnable                bool   // TLS enabled/disabled flag
	TlsCertPath              string // TLS certificate directory path
	K8sJobWorkerImage        string
	K8sJobServiceAccount     string
	K8sJobSSHSecretName      string
	K8sJobTLSSecretName      string
	K8sJobBackoffLimit          int32
	K8sJobTTLSeconds            int32
	K8sJobActiveDeadlineSecs    int64
	MaasApiRetryCount           int
	MaasApiRetryIntervalSeconds int
}

// EnvConfig holds raw environment variable strings for validation
type EnvConfig struct {
	LogLevel                 string `validate:"required" env:"LOG_LEVEL"`
	ServerPort               string `validate:"required" env:"MAAS_SERVER_PORT"`
	VmHostDisk               string `validate:"required" env:"VM_HOST_DISK"`
	LxdListenPort            string `validate:"required" env:"LXD_PORT"`
	SshKey                   string `validate:"required" env:"SSH_KEY"`
	TlsEnable                string `validate:"required" env:"TLS_ENABLE"`
	TlsCertPath              string `validate:"required" env:"TLS_CERT_PATH"`
	K8sJobWorkerImage        string `validate:"required" env:"K8S_JOB_WORKER_IMAGE"`
	K8sJobServiceAccount     string `validate:"required" env:"K8S_JOB_SERVICE_ACCOUNT"`
	K8sJobSSHSecretName      string `validate:"required" env:"K8S_JOB_SSH_SECRET_NAME"`
	K8sJobTLSSecretName      string `env:"K8S_JOB_TLS_SECRET_NAME"`
	K8sJobBackoffLimit          string `validate:"required" env:"K8S_JOB_BACKOFF_LIMIT"`
	K8sJobTTLSeconds            string `validate:"required" env:"K8S_JOB_TTL_SECONDS"`
	K8sJobActiveDeadlineSecs    string `validate:"required" env:"K8S_JOB_ACTIVE_DEADLINE_SECONDS"`
	MaasApiRetryCount           string `validate:"required" env:"MAAS_API_RETRY_COUNT"`
	MaasApiRetryIntervalSeconds string `validate:"required" env:"MAAS_API_RETRY_INTERVAL_SECONDS"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
	configError  error
)

// ResetConfigForTesting resets the global configuration state for testing purposes
// This should only be used in test code
func ResetConfigForTesting() {
	globalConfig = nil
	configOnce = sync.Once{}
	configError = nil
}

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

// loadAndValidateConfig loads environment variables, validates them, and returns parsed Config
func (ev *EnvValidator) loadAndValidateConfig() (*Config, error) {
	// Load raw environment variables
	rawConfig := &EnvConfig{
		LogLevel:                 os.Getenv("LOG_LEVEL"),
		ServerPort:               os.Getenv("MAAS_SERVER_PORT"),
		VmHostDisk:               os.Getenv("VM_HOST_DISK"),
		LxdListenPort:            os.Getenv("LXD_PORT"),
		SshKey:                   os.Getenv("SSH_KEY"),
		TlsEnable:                os.Getenv("TLS_ENABLE"),
		TlsCertPath:              os.Getenv("TLS_CERT_PATH"),
		K8sJobWorkerImage:        os.Getenv("K8S_JOB_WORKER_IMAGE"),
		K8sJobServiceAccount:     os.Getenv("K8S_JOB_SERVICE_ACCOUNT"),
		K8sJobSSHSecretName:      os.Getenv("K8S_JOB_SSH_SECRET_NAME"),
		K8sJobTLSSecretName:      os.Getenv("K8S_JOB_TLS_SECRET_NAME"),
		K8sJobBackoffLimit:          os.Getenv("K8S_JOB_BACKOFF_LIMIT"),
		K8sJobTTLSeconds:            os.Getenv("K8S_JOB_TTL_SECONDS"),
		K8sJobActiveDeadlineSecs:    os.Getenv("K8S_JOB_ACTIVE_DEADLINE_SECONDS"),
		MaasApiRetryCount:           os.Getenv("MAAS_API_RETRY_COUNT"),
		MaasApiRetryIntervalSeconds: os.Getenv("MAAS_API_RETRY_INTERVAL_SECONDS"),
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

	// Parse and validate server port
	serverPort, err := ev.parseAndValidatePort("serverPort", rawConfig.ServerPort)
	if err != nil {
		return nil, err
	}

	// Parse and validate vm host disk
	vmHostDisk, err := ev.parseAndValidateVmHostDisk(rawConfig.VmHostDisk)
	if err != nil {
		return nil, err
	}

	// Parse and validate lxd listen port
	lxdListenPort, err := ev.parseAndValidatePort("lxdListenPort", rawConfig.LxdListenPort)
	if err != nil {
		return nil, err
	}

	// Parse and validate TLS enable flag
	tlsEnable, err := ev.parseAndValidateTlsEnable(rawConfig.TlsEnable)
	if err != nil {
		return nil, err
	}

	k8sBackoffLimit, err := ev.parseRequiredInt32(rawConfig.K8sJobBackoffLimit)
	if err != nil {
		return nil, errors.New("invalid K8S_JOB_BACKOFF_LIMIT: must be integer")
	}
	k8sTTLSeconds, err := ev.parseRequiredInt32(rawConfig.K8sJobTTLSeconds)
	if err != nil {
		return nil, errors.New("invalid K8S_JOB_TTL_SECONDS: must be integer")
	}
	k8sActiveDeadlineSecs, err := ev.parseRequiredInt64(rawConfig.K8sJobActiveDeadlineSecs)
	if err != nil {
		return nil, errors.New("invalid K8S_JOB_ACTIVE_DEADLINE_SECONDS: must be integer")
	}

	maasApiRetryCount, err := ev.parseRequiredInt(rawConfig.MaasApiRetryCount)
	if err != nil {
		return nil, errors.New("invalid MAAS_API_RETRY_COUNT: must be integer")
	}
	maasApiRetryIntervalSeconds, err := ev.parseRequiredInt(rawConfig.MaasApiRetryIntervalSeconds)
	if err != nil {
		return nil, errors.New("invalid MAAS_API_RETRY_INTERVAL_SECONDS: must be integer")
	}

	// Create the parsed config
	config := &Config{
		LogLevel:                 logLevel,
		ServerPort:               serverPort,
		VmHostDisk:               vmHostDisk,
		LxdListenPort:            lxdListenPort,
		SshKey:                   rawConfig.SshKey,
		TlsEnable:                tlsEnable,
		TlsCertPath:              rawConfig.TlsCertPath,
		K8sJobWorkerImage:        rawConfig.K8sJobWorkerImage,
		K8sJobServiceAccount:     rawConfig.K8sJobServiceAccount,
		K8sJobSSHSecretName:      rawConfig.K8sJobSSHSecretName,
		K8sJobTLSSecretName:      rawConfig.K8sJobTLSSecretName,
		K8sJobBackoffLimit:          k8sBackoffLimit,
		K8sJobTTLSeconds:            k8sTTLSeconds,
		K8sJobActiveDeadlineSecs:    k8sActiveDeadlineSecs,
		MaasApiRetryCount:           maasApiRetryCount,
		MaasApiRetryIntervalSeconds: maasApiRetryIntervalSeconds,
	}

	return config, nil
}

func (ev *EnvValidator) parseRequiredInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (ev *EnvValidator) parseRequiredInt32(value string) (int32, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func (ev *EnvValidator) parseRequiredInt64(value string) (int64, error) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
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

// parseAndValidateVmHostDisk validates the VM host disk value
func (ev *EnvValidator) parseAndValidateVmHostDisk(value string) (int, error) {
	message := "invalid vmHostDisk of configuration: value must be integer and 5 (GiB) or greater"

	disk, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(message)
	}

	if disk < 10 || disk > 90 {
		return 0, errors.New(message)
	}

	return disk, nil
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
	case "ServerPort":
		return "MAAS_SERVER_PORT"
	case "VmHostDisk":
		return "VM_HOST_DISK"
	case "LxdListenPort":
		return "LXD_PORT"
	case "SshKey":
		return "SSH_KEY"
	case "TlsEnable":
		return "TLS_ENABLE"
	case "TlsCertPath":
		return "TLS_CERT_PATH"
	case "K8sJobWorkerImage":
		return "K8S_JOB_WORKER_IMAGE"
	case "K8sJobServiceAccount":
		return "K8S_JOB_SERVICE_ACCOUNT"
	case "K8sJobSSHSecretName":
		return "K8S_JOB_SSH_SECRET_NAME"
	case "K8sJobTLSSecretName":
		return "K8S_JOB_TLS_SECRET_NAME"
	case "K8sJobBackoffLimit":
		return "K8S_JOB_BACKOFF_LIMIT"
	case "K8sJobTTLSeconds":
		return "K8S_JOB_TTL_SECONDS"
	case "K8sJobActiveDeadlineSecs":
		return "K8S_JOB_ACTIVE_DEADLINE_SECONDS"
	default:
		return fieldName
	}
}

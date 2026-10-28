package interfaces

// interface of Logging
type Logging interface {
	Init(loggingConfig LoggingConfig) error
	Finalize()
	Write(keyId string, json string) error
}

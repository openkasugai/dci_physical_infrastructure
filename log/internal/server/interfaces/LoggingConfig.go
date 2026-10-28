package interfaces

type LoggingConfig struct {
	LogFile    string
	LogPath    string
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int
}

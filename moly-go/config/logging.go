package config

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level" default:"INFO"`
	Format     string `json:"format" default:"json"`
	FilePath   string `json:"file_path" default:"logs/moly.log"`
	MaxSize    int    `json:"max_size" default:"100"`
	MaxBackups int    `json:"max_backups" default:"10"`
	MaxAge     int    `json:"max_age" default:"30"`
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:      "INFO",
		Format:     "json",
		FilePath:   "logs/moly.log",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
	}
}

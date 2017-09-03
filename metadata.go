package main

import (
	"os"
)

// BuildBaseMetadata constructs a *LogEvent that is designed to be sent
// along with log frames to the collection endpoint. The metadata provides
// additional data that overrides the data present in the log frames.
func BuildBaseMetadata(config *Config) *LogEvent {
	logEvent := NewLogEvent()
	var hostname string

	if config.Hostname != "" {
		hostname = config.Hostname
		logger.Infof("Discovered hostname from config file: %s", hostname)
	} else {
		if os_hostname, err := os.Hostname(); err != nil {
			logger.Warn("Could not autodiscover hostname from operating system")
		} else {
			hostname = os_hostname
			logger.Infof("Discovered hostname from system: %s", hostname)
		}
	}

	logEvent.Context.System.Hostname = hostname

	if !config.CollectEC2MetadataDisabled {
		client := GetEC2Client()
		AddEC2Metadata(client, logEvent)
	} else {
		logger.Info("AWS EC2 metadata collection disabled in config file")
	}

	return logEvent
}

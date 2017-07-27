package main

import (
	"log"
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
	} else {
		if os_hostname, err := os.Hostname(); err != nil {
			log.Println("Could not autodiscover hostname from operating system")
		} else {
			hostname = os_hostname
		}
	}

	logEvent.Context.System.Hostname = hostname

	if !config.CollectEC2MetadataDisabled {
		client := GetEC2Client()
		AddEC2Metadata(client, logEvent)
	}

	return logEvent
}

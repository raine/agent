package main

import (
	"encoding/json"
	"log"
	"os"
)

func BuildMetadata(config *Config) ([]byte, error) {
	log_event_metadata := NewLogEvent()
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

	log_event_metadata.Context.System.Hostname = hostname

	return json.Marshal(log_event_metadata)
}

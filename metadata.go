package main

import (
	"encoding/json"
	"log"
	"os"
)

func BuildMetadata(config *Config) ([]byte, error) {
	log_event := NewLogEvent()
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

	log_event.Context.System.Hostname = hostname

	if !config.CollectEC2MetadataDisabled {
		client := GetEC2Client()
		AddEC2Metadata(client, log_event)
	}

	return json.Marshal(log_event)
}

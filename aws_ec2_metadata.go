package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type EC2Client struct {
	BaseEndpoint string
	HTTPClient   *http.Client
	Timeout      int
}

func AddEC2Metadata(client *EC2Client, logEvent *LogEvent) {
	if !client.Available() {
		log.Println("Agent is not running on an EC2 instance")
		return
	} else {
		log.Println("Agent is running on an EC2 instance")
	}

	amiID, err := client.GetMetadata("ami_id")

	if err != nil {
		log.Println("Could not determine AMI ID the EC2 instance was launched with")
	} else {
		log.Printf("Discovered AMI ID from AWS EC2 metadata: %s", amiID)
	}

	hostname, err := client.GetMetadata("hostname")

	if err != nil {
		log.Println("Cloud not determine the AWS assigned hostname for the EC2 instance")
	} else {
		log.Printf("Discovered EC2 Hostname from AWS EC2 metadata: %s", hostname)
	}

	instanceID, err := client.GetMetadata("instance_id")

	if err != nil {
		log.Println("Could not determine the instance ID for the EC2 instance")
	} else {
		log.Printf("Discovered Instance ID from AWS EC2 metadata: %s", instanceID)
	}

	instanceType, err := client.GetMetadata("instance_type")

	if err != nil {
		log.Println("Could not determine the instance type for the EC2 instance")
	} else {
		log.Printf("Discovered Instance Type from AWS EC2 metadata: %s", instanceType)
	}

	publicHostname, err := client.GetMetadata("public_hostname")

	if err != nil {
		log.Println("Could not determine the AWS assigned public hostname for the EC2 instance")
	} else {
		log.Printf("Discovered EC2 Public Hostname from AWS EC2 metadata: %s", publicHostname)
	}

	logEvent.Context.Platform.AWSEC2.AmiID = amiID
	logEvent.Context.Platform.AWSEC2.Hostname = hostname
	logEvent.Context.Platform.AWSEC2.InstanceID = instanceID
	logEvent.Context.Platform.AWSEC2.InstanceType = instanceType
	logEvent.Context.Platform.AWSEC2.PublicHostname = publicHostname

	return
}

func GetEC2Client() *EC2Client {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	return &EC2Client{
		BaseEndpoint: "http://169.254.169.254",
		HTTPClient:   client,
	}
}

func (client *EC2Client) Available() bool {
	resp, err := client.HTTPClient.Get(client.BaseEndpoint + "/latest/meta-data/")

	if err != nil {
		return false
	}

	resp.Body.Close()

	return true
}

func (client *EC2Client) GetMetadata(field string) (string, error) {
	resp, err := client.HTTPClient.Get(client.BaseEndpoint + "/latest/meta-data/" + field)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body[:]), nil
}

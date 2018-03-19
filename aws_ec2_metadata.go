package main

import (
	"errors"
	"io/ioutil"
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
		logger.Info("Agent is not running on an EC2 instance")
		return
	} else {
		logger.Info("Agent is running on an EC2 instance")
	}

	amiID, err := client.GetMetadata("ami-id")

	if err != nil {
		logger.Warn("Could not determine AMI ID the EC2 instance was launched with")
	} else {
		logger.Infof("Discovered AMI ID from AWS EC2 metadata: %s", amiID)
	}

	hostname, err := client.GetMetadata("hostname")

	if err != nil {
		logger.Warn("Cloud not determine the AWS assigned hostname for the EC2 instance")
	} else {
		logger.Infof("Discovered EC2 Hostname from AWS EC2 metadata: %s", hostname)
	}

	instanceID, err := client.GetMetadata("instance-id")

	if err != nil {
		logger.Warn("Could not determine the instance ID for the EC2 instance")
	} else {
		logger.Infof("Discovered Instance ID from AWS EC2 metadata: %s", instanceID)
	}

	instanceType, err := client.GetMetadata("instance-type")

	if err != nil {
		logger.Warn("Could not determine the instance type for the EC2 instance")
	} else {
		logger.Infof("Discovered Instance Type from AWS EC2 metadata: %s", instanceType)
	}

	publicHostname, err := client.GetMetadata("public-hostname")

	if err != nil {
		logger.Warn("Could not determine the AWS assigned public hostname for the EC2 instance")
	} else {
		logger.Infof("Discovered EC2 Public Hostname from AWS EC2 metadata: %s", publicHostname)
	}

	context := &AWSEC2Context{
		AmiID:          amiID,
		Hostname:       hostname,
		InstanceID:     instanceID,
		InstanceType:   instanceType,
		PublicHostname: publicHostname,
	}

	logEvent.AddEC2Context(context)

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	return true
}

func (client *EC2Client) GetMetadata(field string) (string, error) {
	resp, err := client.HTTPClient.Get(client.BaseEndpoint + "/latest/meta-data/" + field)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("Did not received a valid response for EC2 metadata")
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body[:]), nil
}

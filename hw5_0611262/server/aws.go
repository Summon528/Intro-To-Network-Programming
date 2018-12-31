//+build login

package main

import (
	b64 "encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type instanceStatus struct {
	InstanceID string
	PublicDNS  string
	Online     int
}

var instances = make(map[string]*instanceStatus)
var userInstanceID = make(map[string]string)

func terminateInstance(instanceID string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	svc := ec2.New(sess)
	_, err = svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instanceID},
	})
	if err != nil {
		fmt.Println("Could not terminate instance", err)
		return
	}
	fmt.Println("Terminate", instanceID)
}

func createInstance() (string, string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	svc := ec2.New(sess)
	userData := []string{
		"#!/bin/bash",
		"aws s3 cp s3://nphw5/server /usr/local/bin/nphw5server",
		"chmod +x /usr/local/bin/nphw5server",
		"/usr/local/bin/nphw5server > /var/log/server.log &",
	}

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		LaunchTemplate: &ec2.LaunchTemplateSpecification{
			LaunchTemplateId: aws.String("lt-078b86cf597a45ddd"),
		},
		UserData: aws.String(b64.StdEncoding.EncodeToString([]byte(strings.Join(userData, "\n")))),
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
	})

	if err != nil {
		panic(err)
	}

	instances[*runResult.Instances[0].InstanceId] = &instanceStatus{
		InstanceID: *runResult.Instances[0].InstanceId,
	}

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)
	instanceID := *runResult.Instances[0].InstanceId
	instanceDNS := ""
	for {
		time.Sleep(time.Second * 5)
		if db.Where("id = ?", instanceID).Find(&Instance{}).RecordNotFound() {
			continue
		}
		runResult, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{&instanceID},
		})
		if err != nil {
			panic(err)
		}
		println("Instance", instanceID, "ready")
		instance := instances[instanceID]
		instance.PublicDNS = *runResult.Reservations[0].Instances[0].PublicDnsName
		instanceDNS = instance.PublicDNS
		instance.Online = 1
		return instanceID, instanceDNS
	}
}

func FindEmptyInstance(token string) string {
	for _, instance := range instances {
		if instance.Online < 10 {
			instance.Online++
			userInstanceID[token] = instance.InstanceID
			return instance.PublicDNS
		}
	}
	instanceID, instanceDNS := createInstance()
	userInstanceID[token] = instanceID
	return instanceDNS
}

func LogoutInstance(token string) {
	instance := instances[userInstanceID[token]]
	instance.Online--
	delete(userInstanceID, token)
	if instance.Online == 0 {
		terminateInstance(instance.InstanceID)
		db.Delete(Instance{}, "id = ?", instance.InstanceID)
		delete(instances, instance.InstanceID)
	}
	return
}

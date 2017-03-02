#!/bin/bash

GOOS=linux GOARCH=amd64 go build .
mv elb_health_check elb_health_check_linux_amd64
GOOS=linux GOARCH=386 go build .
mv elb_health_check elb_health_check_linux_386
GOOS=darwin GOARCH=amd64 go build .
mv elb_health_check elb_health_check_darwin_amd64

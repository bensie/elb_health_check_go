.PHONY: build
build:
	go build -o elb_health_check .

.PHONY: release
release:
	GOOS=linux GOARCH=amd64 $(MAKE) build
	mv elb_health_check elb_health_check_linux_amd64
	GOOS=linux GOARCH=arm64 $(MAKE) build
	mv elb_health_check elb_health_check_linux_arm64
	GOOS=darwin GOARCH=amd64 $(MAKE) build
	mv elb_health_check elb_health_check_darwin_amd64

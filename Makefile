SHELL := /bin/bash

# hopefully this module disabling nastyness will go away with 1.14...
devsetup:
	GO111MODULE=off go get "github.com/kisielk/errcheck"
	GO111MODULE=off go get "golang.org/x/lint/golint"
	GO111MODULE=off go get "github.com/gordonklaus/ineffassign"
	sudo apt install gcc-arm-linux-gnueabi

checkfmt:
	! gofmt -l -d -s . 2>&1 | read

checkvet:
	go vet ./...

checkiea:
	ineffassign .

checkerrs:
	errcheck -blank -asserts -ignoretests ./...

checklint:
	golint -set_exit_status -min_confidence 0.81 ./...

test:
	go test -race -cover ./...

validate: test checkfmt checkerrs checkvet checkiea checklint

build:
	env CC=arm-linux-gnueabi-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build -o weather-station ./cmd/weather_station/

deploy: build
	scp -r migrations/ weather-station ${STNUSER}@${STNHOST}:~/

.PHONY: checkfmt checkvet checkiea checkerrs checklint test validate build deploy

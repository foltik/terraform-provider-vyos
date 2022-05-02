HOSTNAME=github.com
NAMESPACE=foltik
NAME=vyos
BINARY=terraform-provider-${NAME}
VERSION=0.2.0
OS_ARCH=linux_amd64

default: install

doc:
	go generate

build:
	go build -o ${BINARY}

release:
	GOOS=linux   GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=linux   GOARCH=arm64 go build -o ./bin/${BINARY}_${VERSION}_linux_arm64
	GOOS=linux   GOARCH=arm   go build -o ./bin/${BINARY}_${VERSION}_linux_arm
	GOOS=darwin  GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=darwin  GOARCH=arm64 go build -o ./bin/${BINARY}_${VERSION}_darwin_arm64
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test:
	go test ./...

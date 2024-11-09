clean:
	rm -rf bin && mkdir bin
	rm -rf artifacts && mkdir artifacts

build: clean
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bin/bluesky/bootstrap ./cmd/bluesky

package: build
	@cd bin/bluesky && zip -r9 ../../artifacts/bluesky.zip bootstrap

deploy: package
	aws lambda update-function-code --function-name AWS-News-Update-eu-central-1 --zip-file fileb://./artifacts/bluesky.zip

local-build:
	go build -tags lambda.norpc -o bin/bluesky/bootstrap ./cmd/bluesky

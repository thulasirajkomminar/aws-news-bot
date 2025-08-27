# Configuration
AWS_REGION ?= eu-central-1
LAMBDA_FUNCTION ?= AWS-News-Bot-$(AWS_REGION)

clean:
	rm -rf bin && mkdir bin
	rm -rf artifacts && mkdir artifacts

build: clean
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc \
		-ldflags="-X main.Version=$$(git describe --tags --always --dirty)" \
		-o bin/awsnewsbot/bootstrap ./cmd/awsnewsbot

package: build
	@cd bin/awsnewsbot && zip -r9 ../../artifacts/awsnewsbot.zip bootstrap

deploy: package
	@echo "Deploying to $(LAMBDA_FUNCTION) in $(AWS_REGION)..."
	@aws lambda update-function-code \
		--region $(AWS_REGION) \
		--function-name $(LAMBDA_FUNCTION) \
		--zip-file fileb://./artifacts/awsnewsbot.zip
	@echo "Waiting for function update..."
	@aws lambda wait function-updated \
		--region $(AWS_REGION) \
		--function-name $(LAMBDA_FUNCTION)
	@echo "Deployment completed successfully"

local-build:
	@CGO_ENABLED=0 go build -tags lambda.norpc \
		-ldflags="-X main.Version=$$(git describe --tags --always --dirty)" \
		-o bin/awsnewsbot/bootstrap ./cmd/awsnewsbot

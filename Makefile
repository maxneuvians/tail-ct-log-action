.PHONY: dev release release-test

dev:
	@echo "Running dev environment"
	@DOMAIN=example.com go run main.go

release:
	@mkdir -p release/latest
	@docker build -t tail-ct-log-action-build -f Dockerfile.build .
	@docker create -ti --name tail-ct-log-action-build tail-ct-log-action-build bash 
	@docker cp tail-ct-log-action-build:/tail-ct-log-action release/latest/tail-ct-log-action
	@docker rm -f tail-ct-log-action-build

release-test:
	@docker build -t tail-ct-log-action-build -f Dockerfile.build .
	@docker create -ti --name tail-ct-log-action-build tail-ct-log-action-build bash 
	@docker rm -f tail-ct-log-action-build
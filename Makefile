.PHONY: start-nats
## Starts nats-streaming server
start-nats:
	@docker-compose up --no-start nats
	docker-compose start nats

.PHONY: stop-nats
## Stops the nats-streaming server
stop-nats:
	docker-compose stop nats

.PHONY: test-nonats
## runs go test, expects nats to be running
test-nonats:
	@go clean -testcache
	@which gotest || go get -u github.com/rakyll/gotest
	@gotest -p 1 -v -mod=vendor $$(go list ./... | grep -v /vendor/)

.PHONY: test
## runs go test, starts and stops nats
test: vendor start-nats
	@bash -c "trap 'trap - SIGINT SIGTERM ERR; docker-compose stop nats; exit 1' SIGINT SIGTERM ERR; $(MAKE) test-nonats"
	@$(MAKE) stop-nats

####################
# Helpers and misc #
####################

# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: tidy
## Runs go mod tidy
tidy:
	@go mod tidy

.PHONY: vendor
## Updates vendored deps
vendor:
	@echo "updating vendored deps..."
	@go mod vendor
	@echo "done!"

.PHONY: help
# Help target stolen from this comment: https://gist.github.com/prwhite/8168133#gistcomment-2278355
## Show help
help:
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

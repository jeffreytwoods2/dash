include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/web: run the development server
.PHONY: run/web
run/web:
	@go run ./cmd/web -port=4000 -db-dsn=${DASH_DB_DSN}

## run/prod: run the production server
.PHONY: run/prod
run/prod:
	@echo 'Starting production server...'
	@./bin/web -port=4000 -db-dsn=${DASH_DB_DSN} -env=production

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/web: build the cmd/web application
.PHONY: build/web
build/web:
	@echo 'Building cmd/web...'
	@go build -ldflags="-s -w" -o=./bin/web ./cmd/web
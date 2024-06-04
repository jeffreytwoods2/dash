include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/web: run the development server
.PHONY: run/web
run/web:
	@go run ./cmd/web -port=4000 -db-dsn=${DASH_DB_DSN}

## db/new name=$1: create a new database migration
.PHONY: db/new
db/new:
	@echo 'Creating migration files for ${name}...'
	@migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/up: apply all up database migrations
.PHONY: db/up
db/up: confirm
	@echo 'Running up migrations...'
	@migrate -path ./migrations -database ${DASH_DB_DSN} up

## db/down: apply all down database migrations
.PHONY: db/down
db/down:
	@echo 'Running down migrations...'
	@migrate -path ./migrations -database ${DASH_DB_DSN} down

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/web: build the cmd/web application
.PHONY: build/web
build/web:
	@echo 'Building cmd/web...'
	@go build -ldflags="-s -w" -o=./bin/web ./cmd/web
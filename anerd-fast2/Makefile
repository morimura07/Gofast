####################################################
# Docker
####################################################

start:
	docker compose -f docker-compose.yml up --build
startc:
	docker compose -f docker-compose.yml -f docker-compose.client.yml up --build
startm:
	docker compose -f docker-compose.yml -f docker-compose.client.yml -f docker-compose.otel.yml up --build
startcm:
	docker compose -f docker-compose.yml -f docker-compose.otel.yml up --build
stop:
	docker ps -aq | xargs -r docker stop
down:
	docker ps -aq | xargs -r docker rm -f

####################################################
# Database
####################################################

migrate:
	cd app/service-core/storage && goose -dir migrations postgres "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable" up

####################################################
# Format
####################################################

format:
	cd app && go fmt ./...
	cd app/service-core && go fmt ./...
	cd app/service-client && npm run format

####################################################
# Codegen
####################################################

gen:
	buf generate

sql:
	cd app/service-core/storage && rm -rf query && sqlc generate -f sqlc.yaml

keys:
	openssl genpkey -algorithm Ed25519 -out app/private.pem
	openssl pkey -in app/private.pem -pubout -out app/public.pem

####################################################
# External Services
####################################################

stripe:
	stripe listen --latest --forward-to localhost:4000/payments/webhook

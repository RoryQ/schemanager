# The following targets are expected to be run from the root Makefile so the paths are relative to the project root.
# The migrate program expects a DATABASE_CONN environment variable set with the DSN.

.PHONY: migrate-new
migrate-new:
	go run db/migrate.go new

.PHONY: migrate
migrate:
	go run db/migrate.go up
	go run db/migrate.go status

# Run `make schema` to export the final schema of the applied migrations
.PHONY: schema
schema: _schema-pg-up
	go run db/migrate.go up
	go run db/migrate.go dump
	@make _schema-pg-down

# recreates usage section in readme if there are changes.
db/README.md: db/migrate.go
	go run db/usage.go

# schema helpers

.PHONY: _schema-pg-up
_schema-pg-up:
	-@make _schema-pg-down # clear previous
	@docker run --rm --detach -p 5432 -e POSTGRES_PASSWORD=pwd --name pg-schema postgres:15
	@sleep 5 # TODO wait until ready


.PHONY: _schema-pg-down
_schema-pg-down:
	-@docker stop pg-schema 2>&1>/dev/null || true

SCHEMA_PG_HOST = $(shell docker port pg-schema 5432 | sed 's/0.0.0.0/localhost/')

schema: export DATABASE_CONN=postgres://postgres:pwd@$(SCHEMA_PG_HOST)/postgres?sslmode=disable
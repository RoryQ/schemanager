# example dockerfile can be used as a separate deployment image or the binary can be included in the app image
# and invoked pre-deployment
FROM golang
WORKDIR /app

COPY db db
COPY go.mod go.sum ./
RUN go build db/migrate.go

FROM debian
COPY --from=0 /app/migrate .

# set DATABASE_CONN environment variable
ENTRYPOINT ./migrate up
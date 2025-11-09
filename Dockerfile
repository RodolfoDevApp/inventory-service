# build
FROM golang:1.22-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin/inventory-service ./cmd/inventory-service

# runtime
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /bin/inventory-service /app/inventory-service

ENV HTTP_PORT=8085
EXPOSE 8085

ENTRYPOINT ["/app/inventory-service"]

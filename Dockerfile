# ---------- build ----------
FROM golang:1.22-alpine AS build

# IMPORTANTE para que no pida 1.25.x dentro del contenedor
ENV GOTOOLCHAIN=local

WORKDIR /src

# solo mod/sum primero para cachear dependencias
COPY go.mod go.sum ./
RUN go mod download

# ahora sí, el código
COPY . .

# construimos el binario
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /inventory-service ./cmd/inventory-service

# ---------- runtime ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=build /inventory-service /app/inventory-service

ENV HTTP_PORT=8085
EXPOSE 8085

ENTRYPOINT ["/app/inventory-service"]

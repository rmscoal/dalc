# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o server-app ./server/main.go

# Stage 2: Create the final image
FROM alpine:latest

COPY --from=builder /app/server-app /src/server-app

ENTRYPOINT ["/src/server-app"]

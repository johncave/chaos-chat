# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o chatapp main.go

# Final image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/chatapp ./chatapp
COPY static ./static
EXPOSE 8080
CMD ["./chatapp"] 
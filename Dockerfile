FROM golang:1.26.6-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/projecthub ./cmd/api

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && adduser -D app
WORKDIR /app
COPY --from=builder /out/projecthub /app/projecthub
ENV TZ=Asia/Jakarta
USER app
EXPOSE 8080
CMD ["/app/projecthub"]

FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /url-shortener ./cmd/url-shortener

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /url-shortener ./url-shortener

EXPOSE 8080

ENTRYPOINT ["./url-shortener"]

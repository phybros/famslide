FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /famslide ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /famslide /usr/local/bin/famslide
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/famslide"]

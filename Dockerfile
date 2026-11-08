FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
FROM alpine:3.20
WORKDIR /app
COPY --from=build /server /server
COPY templates ./templates
EXPOSE 8080
ENTRYPOINT ["/server"]

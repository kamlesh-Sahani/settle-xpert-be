FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && adduser -D app
WORKDIR /
USER app
COPY --from=build /app /app
COPY --from=build /src/assets /assets
EXPOSE 8080
CMD ["/app"]

FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY libs/platform libs/platform
COPY services/placement-service services/placement-service
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /service ./services/placement-service/cmd
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
COPY --from=build /service /service
USER app
EXPOSE 8080
ENTRYPOINT ["/service"]

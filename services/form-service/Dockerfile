FROM golang:1.21 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x && go mod verify
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -v -o app .

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/app .
CMD ["./app"]

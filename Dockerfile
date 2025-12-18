# Build
FROM golang:1.25.5-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/restic-browser .

# Runtime (restic + ca-certs)
FROM alpine:3.23
RUN apk add --no-cache ca-certificates restic
WORKDIR /app
COPY --from=build /out/restic-browser /app/restic-browser
EXPOSE 8080
ENTRYPOINT ["/app/restic-browser"]

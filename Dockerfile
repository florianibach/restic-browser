# Build
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/restic-ui .

# Runtime (restic + ca-certs)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates restic
WORKDIR /app
COPY --from=build /out/restic-ui /app/restic-ui
EXPOSE 8080
ENTRYPOINT ["/app/restic-ui"]

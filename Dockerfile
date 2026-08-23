FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/neurons-server ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S neurons && adduser -S neurons -G neurons

WORKDIR /app

COPY --from=build /out/neurons-server ./neurons-server
COPY migrations ./migrations

RUN mkdir -p /app/uploads && chown -R neurons:neurons /app

USER neurons

EXPOSE 8080

ENTRYPOINT ["./neurons-server"]

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -g 65532 -S application \
    && adduser -u 65532 -S -G application application

COPY --from=build /out/backend /backend

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/backend"]

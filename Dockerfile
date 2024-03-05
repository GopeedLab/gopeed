FROM instrumentisto/flutter:3.16.8 AS flutter
WORKDIR /app
COPY ./ui/flutter/pubspec.yaml ./ui/flutter/pubspec.lock ./
RUN flutter pub get
COPY ./ui/flutter ./
RUN flutter build web --web-renderer html

FROM golang:1.21.6 AS go
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
COPY --from=flutter /app/build/web ./cmd/web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -tags nosqlite,web -ldflags="-s -w \
      -X github.com/GopeedLab/gopeed/pkg/base.Version=${VERSION}" \
      -X github.com/GopeedLab/gopeed/pkg/base.InDocker=${IN_DOCKER}" \
      -o dist/gopeed github.com/GopeedLab/gopeed/cmd/web

FROM alpine:3.14.2
LABEL maintainer="monkeyWie"
WORKDIR /app
COPY --from=go /app/dist/gopeed ./
VOLUME ["/app/storage"]
EXPOSE 9999
ENTRYPOINT ["./gopeed"]

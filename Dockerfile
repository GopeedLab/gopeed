FROM cirrusci/flutter:3.3.7 AS flutter
WORKDIR /app
COPY ./ui/flutter/pubspec.yaml ./ui/flutter/pubspec.lock ./
RUN flutter pub get
COPY ./ui/flutter ./
RUN flutter build web

FROM golang:1.19.3 AS go
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
COPY --from=flutter /app/build/web/* ./cmd/web/dist
RUN go build -tags nosqlite,web -ldflags="-s -w" -o dist/gopeed github.com/monkeyWie/gopeed/cmd/web

FROM alpine:3.14.2
LABEL maintainer="monkeyWie"
WORKDIR /app
COPY --from=go /app/dist/gopeed ./
EXPOSE 9999
ENTRYPOINT ["./gopeed"]
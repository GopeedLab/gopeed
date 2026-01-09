FROM golang:1.24.0 AS go
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -tags nosqlite,web \
      -ldflags="-s -w -X github.com/GopeedLab/gopeed/pkg/base.Version=$VERSION -X github.com/GopeedLab/gopeed/pkg/base.InDocker=true" \
      -o dist/gopeed github.com/GopeedLab/gopeed/cmd/web

FROM alpine:3.18
LABEL maintainer="monkeyWie"
WORKDIR /app
COPY --from=go /app/dist/gopeed ./
COPY entrypoint.sh ./entrypoint.sh
RUN apk update && \
    apk add --no-cache su-exec ; \
    chmod +x ./entrypoint.sh && \
    rm -rf /var/cache/apk/*
VOLUME ["/app/storage"]
ENV PUID=0 PGID=0 UMASK=022
EXPOSE 9999
ENTRYPOINT ["./entrypoint.sh"]

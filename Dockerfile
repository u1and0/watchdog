# Golang build container
FROM golang:1.23.2-alpine3.20 AS gobuilder
WORKDIR /work
# main go
WORKDIR /work
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY ./main.go .
RUN go build # -a -ldflags '-linkmode external -extldflags "-static"'

FROM gcr.io/distroless/static-debian12 AS runner
USER nonroot
# ENV SLACK_WEBHOOK=
#     SLACK_USER=
#     MIN_INTERVAL=2s
#     MAX_INTERVAL=16s
COPY --from=gobuilder /work/watchdog /usr/bin/watchdog
ENTRYPOINT ["/usr/bin/watchdog"]

LABEL maintainer="u1and0 <e01.ando60@gmail.com>" \
      description="特定のエンドポイントを監視して、ステータス200を取得できなければslackへ異常通知します。" \
      version="watchdog v0.1.0" \
      usage="SLACK_WEBHOOK=https://xxxx SLACK_USER=abcde docker run -t --rm u1and0/watchdog\
            -e http://localhost:8080/index\
            -m 2s\
            -M 16s"

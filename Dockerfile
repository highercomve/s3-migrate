FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN apk update; apk add -U --no-cache \
	git \
	curl \
	build-base

RUN go get -d -v ./... \
	&& go install -v ./...

FROM alpine

RUN apk update; apk add ca-certificates

COPY --from=builder /go/bin/s3-migrate /usr/bin/

ENTRYPOINT [ "/usr/bin/s3-migrate" ]

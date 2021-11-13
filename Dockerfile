########################################
# stage: build
########################################
FROM golang:1.15.14-alpine3.14 as builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . /build

WORKDIR /build/cmd

RUN go build -o subscan

########################################
# stage: final
########################################
FROM python:3.9-alpine

COPY --from=builder /build/cmd/subscan /app/subscan

WORKDIR /app

RUN apk add --no-cache tini=0.19.0-r0

COPY configs    /app/configs
COPY cmd/run.py /app/run.py

ENTRYPOINT ["/sbin/tini", "--"]

CMD ["/app/subscan"]

EXPOSE 4399

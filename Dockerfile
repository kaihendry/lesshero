FROM golang:1.27 AS build-env

WORKDIR /workspace

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

COPY . .

RUN go build

FROM scratch

COPY --from=build-env /workspace/lesshero /

ENTRYPOINT ["/lesshero"]

FROM golang:latest AS build-env

RUN mkdir -p /workspace
WORKDIR /workspace

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

COPY . .

RUN go build

FROM scratch

COPY --from=build-env /workspace/lesshero /

ENTRYPOINT ["/lesshero", "-o", "lesshero.html"]

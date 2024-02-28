FROM golang:1.22 AS build-env

RUN mkdir -p /workspace
WORKDIR /workspace

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build

FROM scratch

COPY --from=build-env /workspace/lesshero /

ENTRYPOINT ["/lesshero", "-c", "lesshero.html"]

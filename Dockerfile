FROM golang:1.15 AS build
WORKDIR /src
COPY go.mod /src/go.mod
COPY go.sum /src/go.sum
RUN go mod download
COPY . /src
RUN make build

FROM python:2.7

RUN apt update && apt install -y curl psmisc net-tools
RUN curl -s https://pagekite.net/pk/ |sh
COPY --from=build /src/dist/pk-ingress-controller /
COPY --from=build /src/src/template/pagekite.rc.tmpl /src/template/pagekite.rc.tmpl

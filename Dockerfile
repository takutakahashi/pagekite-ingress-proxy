FROM golang:1.12 AS build
COPY . /src
WORKDIR /src
RUN make build

FROM python:2.7

RUN apt update && apt install -y curl
RUN curl -s https://pagekite.net/pk/ |bash
COPY --from=build /src/dist/pk-ingress-controller /

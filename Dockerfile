FROM golang:1.13.4 AS build
COPY . /src
WORKDIR /src
RUN make build

FROM python:2.7

RUN apt update && apt install -y curl
RUN curl -s https://pagekite.net/pk/ |sh
COPY --from=build /src/dist/pk-ingress-controller /
COPY --from=build /src/src/template/pagekite.rc.tmpl /src/template/pagekite.rc.tmpl

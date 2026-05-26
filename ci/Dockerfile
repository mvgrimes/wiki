# syntax=docker/dockerfile:1

FROM golang:1.26-alpine as build

# Add Tini
ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini

RUN adduser -S -D -H -h /app appuser
WORKDIR /app

COPY ./wiki-linux-amd64 /app/wiki
RUN chmod +x /app/wiki

USER appuser

ENTRYPOINT [ "/tini", "--" ]
CMD ./wiki bootstrap && ./wiki server

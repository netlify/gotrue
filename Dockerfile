FROM golang:1.9.2
WORKDIR /go/src/github.com/netlify/gotrue
COPY . /go/src/github.com/netlify/gotrue/
RUN make deps build

FROM alpine:3.7
RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 netlify && mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=0 /go/src/github.com/netlify/gotrue/gotrue /usr/local/bin/gotrue
COPY --from=0 /go/src/github.com/netlify/gotrue/migrations /usr/local/etc/gotrue/migrations/
RUN chown netlify:netlify /usr/local/bin/gotrue && chown -R netlify:netlify /usr/local/etc/gotrue

USER netlify
ENV GOTRUE_DB_MIGRATIONS_PATH /usr/local/etc/gotrue/migrations
CMD ["gotrue"]

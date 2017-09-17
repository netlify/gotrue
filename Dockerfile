FROM alpine:latest  
RUN apk --no-cache add ca-certificates
RUN adduser -D -u 1000 netlify && mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY gotrue /usr/local/bin/
USER netlify
WORKDIR /home/netlify
CMD ["gotrue"]

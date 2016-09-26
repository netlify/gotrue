FROM calavera/go-glide:v0.12.2

ADD . /go/src/github.com/netlify/netlify-auth

RUN useradd -m netlify && cd /go/src/github.com/netlify/netlify-auth && make deps build && mv netlify-auth /usr/local/bin/

USER netlify
CMD ["netlify-auth"]

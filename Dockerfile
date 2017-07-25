FROM netlify/go-glide:v0.12.3

ADD . /go/src/github.com/netlify/gotrue

RUN useradd -m netlify && cd /go/src/github.com/netlify/gotrue && make deps build && mv gotrue /usr/local/bin/

USER netlify
CMD ["gotrue"]

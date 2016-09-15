FROM calavera/go-glide:v0.12.2

ADD . /go/src/github.com/netlify/authlify

RUN cd /go/src/github.com/netlify/authlify && make deps build && mv authlify /usr/local/bin/

CMD ["authlify"]

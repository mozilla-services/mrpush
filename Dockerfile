FROM golang:1

ENV GO15VENDOREXPERIMENT=1

ADD . /go/src/github.com/jasonthomas/mrpush
RUN go install github.com/jasonthomas/mrpush


CMD ["mrpush"]

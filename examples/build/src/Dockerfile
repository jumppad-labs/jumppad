FROM golang:latest as build

WORKDIR /go/src/build

COPY . /go/src/build 

RUN CGO_ENABLED=0 go build -o /bin/app main.go

FROM alpine:latest

COPY --from=build /bin/app /bin/app
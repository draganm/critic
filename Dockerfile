FROM golang:1.15.1-alpine3.12 as build
RUN mkdir /critic
WORKDIR /critic
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build  -o critic .

FROM alpine:3.14.3
RUN apk --update add ca-certificates
COPY --from=build /critic/critic /critic
ENTRYPOINT ["/critic"]
EXPOSE 3001

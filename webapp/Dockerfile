FROM golang:1.16-alpine as builder

ARG GITHUB_USER
ARG GITHUB_ACCESS_TOKEN

RUN apk add --no-cache ca-certificates git

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534' > /user/group

RUN printf "\
    machine github.com\n\
    login ${GITHUB_USER}\n\
    password ${GITHUB_ACCESS_TOKEN}\n\
    \n\
    machine api.github.com\n\
    login ${GITHUB_USER}\n\
    password ${GITHUB_ACCESS_TOKEN}\n"\
    >> /root/.netrc

RUN chmod 600 /root/.netrc

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir /config
RUN mkdir /tls

RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /bin/maths-webapp .


FROM scratch

COPY --from=builder /bin/maths-webapp /maths-webapp
COPY --from=builder /src/i18n/messages /i18n/messages
COPY --from=builder /src/static /static
COPY --from=builder /src/templates /templates

COPY --from=builder /config /config
COPY --from=builder /tls /tls

COPY --from=builder /user/group /user/passwd /etc/

USER nobody:nobody

EXPOSE 8043

ENTRYPOINT ["/maths-webapp", "/config/config.json"]
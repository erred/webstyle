FROM golang:alpine AS build

WORKDIR /workspace
ENV CGO_ENABLED=0
COPY . .
RUN go build -o /bin/webrender ./cmd/webrender

FROM scratch

COPY --from=build /bin/webrender /bin/

ENTRYPOINT [ "/bin/webrender" ]

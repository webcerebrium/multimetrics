FROM golang:alpine
WORKDIR /app
ADD go.mod ./
ADD main.go ./
RUN ls -All && go build -o ./bin/multimetrics .

FROM alpine
COPY --from=0 /app/bin/multimetrics /app/multimetrics
CMD ["/app/multimetrics"]
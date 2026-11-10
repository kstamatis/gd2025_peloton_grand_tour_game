FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY server /app/server
COPY public /app/public

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/server"]

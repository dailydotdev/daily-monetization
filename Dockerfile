FROM binxio/gcp-get-secret

FROM alpine
RUN apk update && \
   apk add ca-certificates && \
   update-ca-certificates && \
   rm -rf /var/cache/apk/*

EXPOSE 9090

COPY --from=0 /gcp-get-secret /usr/local/bin/

ADD ip2location /ip2location
ADD main /
ENTRYPOINT ["/usr/local/bin/gcp-get-secret"]
CMD ["/main"]

FROM alpine
RUN apk update && \
   apk add ca-certificates && \
   update-ca-certificates && \
   rm -rf /var/cache/apk/*

EXPOSE 9090

ADD tmpl /tmpl
ADD main /
CMD ["/main"]
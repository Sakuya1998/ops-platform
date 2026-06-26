FROM apache/apisix:3.9.0-alpine

COPY config.yaml /usr/local/apisix/conf/config.yaml
COPY routes.yaml /usr/local/apisix/conf/apisix.yaml

EXPOSE 9080 9443 9180

CMD ["apisix", "start"]

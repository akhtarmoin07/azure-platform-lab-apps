FROM nginx:1.28-alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY public/ /usr/share/nginx/html/

RUN apk add --no-cache --upgrade \
        "libcrypto3>=3.5.6-r0" \
        "libssl3>=3.5.6-r0" \
    && chown -R 101:101 /usr/share/nginx/html

USER 101:101
EXPOSE 8080

ENTRYPOINT ["nginx", "-g", "daemon off;"]

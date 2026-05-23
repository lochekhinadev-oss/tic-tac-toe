FROM nginx:1.27-alpine

COPY gateway/nginx.conf /etc/nginx/conf.d/default.conf
COPY gateway/proxy_headers.conf /etc/nginx/conf.d/proxy_headers.conf
COPY gateway/routes /etc/nginx/conf.d/routes

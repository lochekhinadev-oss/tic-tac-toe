FROM nginx:1.27-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY frontend/index.html /usr/share/nginx/html/index.html
COPY frontend/style.css /usr/share/nginx/html/style.css
COPY frontend/app.js /usr/share/nginx/html/app.js
COPY frontend/favicon.svg /usr/share/nginx/html/favicon.svg
COPY frontend/favicon.ico /usr/share/nginx/html/favicon.ico

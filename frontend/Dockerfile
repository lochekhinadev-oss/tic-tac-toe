FROM nginx:1.27-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY frontend/index.html /usr/share/nginx/html/index.html
COPY frontend/swagger.html /usr/share/nginx/html/swagger.html
COPY frontend/style.css /usr/share/nginx/html/style.css
COPY frontend/app.js /usr/share/nginx/html/app.js
COPY backend/tic-tac-toe/docs/swagger.yaml /usr/share/nginx/html/openapi.yaml

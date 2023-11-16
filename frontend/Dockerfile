FROM nginx:alpine
COPY . /usr/share/nginx/html
ARG API_ENDPOINT
RUN sed -i 's/localhost:9000/'"$API_ENDPOINT"'/g' /usr/share/nginx/html/js/main.js

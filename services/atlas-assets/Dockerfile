FROM nginx:1.29-alpine

COPY nginx.conf /etc/nginx/nginx.conf

VOLUME /usr/assets

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]

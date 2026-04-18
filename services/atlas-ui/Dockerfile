FROM node:24-alpine AS builder

WORKDIR /build

COPY services/atlas-ui/package.json services/atlas-ui/package-lock.json ./
RUN npm ci

COPY services/atlas-ui/ .
RUN npm run build

FROM nginx:alpine

COPY --from=builder /build/dist /usr/share/nginx/html
COPY services/atlas-ui/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

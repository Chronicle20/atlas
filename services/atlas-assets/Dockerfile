FROM nginx:1.31-alpine

# Top-level nginx.conf is namespace-agnostic; the server block lives in a
# template that the upstream image's docker-entrypoint envsubst step
# materialises to /etc/nginx/conf.d/default.conf at pod startup, expanding
# ${POD_NAMESPACE} (downward-API-injected) so atlas-wz-extractor resolves
# to the correct per-environment namespace.
#
# The base image ships its own /etc/nginx/conf.d/default.conf placeholder
# that would collide with our rendered default.conf — remove it.
RUN rm -f /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/nginx.conf
COPY default.conf.template /etc/nginx/templates/default.conf.template

VOLUME /usr/assets

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]

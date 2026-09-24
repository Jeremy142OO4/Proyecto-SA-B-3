FROM node:24-alpine AS compilacion
WORKDIR /aplicacion
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.29-alpine
ENV BACKEND_URL=http://api-gateway:8080
ENV PORT=80
ENV NGINX_ENVSUBST_FILTER="BACKEND_URL|PORT"
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=compilacion /aplicacion/dist /usr/share/nginx/html
EXPOSE 80

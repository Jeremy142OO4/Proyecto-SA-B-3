FROM node:24-alpine AS compilacion
WORKDIR /aplicacion
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.29-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=compilacion /aplicacion/dist /usr/share/nginx/html
EXPOSE 80

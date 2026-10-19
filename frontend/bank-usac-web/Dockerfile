FROM node:24-alpine AS compilacion
WORKDIR /aplicacion
COPY package.json ./
RUN npm install
COPY . .
# Contrato original inmutable; el contexto de esta imagen solo incluye el frontend.
ADD --checksum=sha256:4888b951aa48f95d8d301098c8aa642adfaa0e45a380975b8ce42acf3506df6a https://raw.githubusercontent.com/Jeremy142OO4/Proyecto-SA-B-3/181b3167e91297f8fb16434ec2c29b7f75aad4f3/contracts/interbank/openapi.yaml /contracts/interbank/openapi.yaml
RUN npm run build

FROM nginx:1.29-alpine
ENV BACKEND_URL=http://api-gateway:8080
ENV PORT=80
ENV NGINX_ENVSUBST_FILTER="BACKEND_URL|PORT"
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=compilacion /aplicacion/dist /usr/share/nginx/html
EXPOSE 80

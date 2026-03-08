FROM node:20-alpine AS builder

WORKDIR /app

ARG VITE_APP_ENV=local
ARG VITE_API_BASE_URL=/api
ENV VITE_APP_ENV=$VITE_APP_ENV
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:alpine

COPY --from=builder --chown=nginx:root /app/dist /usr/share/nginx/html
COPY --chown=nginx:root nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

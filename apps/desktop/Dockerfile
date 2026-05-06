FROM node:22-bookworm-slim AS build

WORKDIR /src
COPY package.json package-lock.json ./
COPY apps/desktop/package.json apps/desktop/package.json
COPY packages/shared/package.json packages/shared/package.json
COPY packages/ui/package.json packages/ui/package.json
RUN npm ci

COPY apps/desktop apps/desktop
COPY packages packages

ARG VITE_FORGE_API_URL=http://127.0.0.1:18492
ENV VITE_FORGE_API_URL=$VITE_FORGE_API_URL
RUN npm -w @forge/desktop run build

FROM nginx:1.27-alpine

COPY apps/desktop/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/apps/desktop/dist /usr/share/nginx/html

EXPOSE 80

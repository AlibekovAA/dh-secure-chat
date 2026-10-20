FROM node:22.22.2-alpine AS builder

WORKDIR /app

# npm 12.2.0 requires promise-retry at runtime but does not install it itself.
RUN npm install --global npm@12.2.0 promise-retry@2.0.1

COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./
RUN npm ci

COPY . .

RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

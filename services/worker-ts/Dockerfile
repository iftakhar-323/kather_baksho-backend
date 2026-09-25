# Stage 1: Build TypeScript into JavaScript
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install

COPY tsconfig.json ./
COPY src ./src

RUN npm run build

# Stage 2: Production runtime
FROM node:20-alpine

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=8083

COPY package.json package-lock.json* ./
RUN npm install --omit=dev

COPY --from=builder /app/dist ./dist

EXPOSE 8083

CMD ["node", "dist/server.js"]


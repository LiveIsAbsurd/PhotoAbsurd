# ==============================================================================
# Multi-stage Dockerfile: Сборка React фронтенда и запуск в Nginx
# ==============================================================================

# 1. Этап сборки (Node.js)
FROM node:20-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей для эффективного кэширования слоёв Docker
COPY frontend/package*.json ./
RUN npm ci || npm install

# Копируем исходный код фронтенда
COPY frontend/ ./

# Компилируем продакшен-бандл в директорию /app/dist
RUN npm run build

# 2. Финальный этап (Nginx)
FROM nginx:alpine

# Копируем собранный фронтенд в стандартную директорию Nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# Копируем конфигурации веб-сервера
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

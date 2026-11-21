# 📷 PhotoGallery (PhotoService)

<p align="center">
  <b>Минималистичная, высокопроизводительная веб-галерея для авторской фотографии</b><br>
  Фокус на эстетике, скорости доставки контента, строгой защите приватности и чистоте визуального восприятия.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-1.2.0-black?style=for-the-badge" alt="Version 1.2.0">
  <img src="https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.22">
  <img src="https://img.shields.io/badge/React-18.3-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React 18">
  <img src="https://img.shields.io/badge/Vite-5.4-646CFF?style=for-the-badge&logo=vite&logoColor=white" alt="Vite 5">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL 16">
  <img src="https://img.shields.io/badge/Nginx-Alpine-009639?style=for-the-badge&logo=nginx&logoColor=white" alt="Nginx">
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
</p>

---

## ✨ Ключевые возможности

- 🖼️ **Автоматический пайплайн обработки фото:**
  - Конвертация загружаемых снимков любого формата (JPEG, PNG) в современный сжатый формат **WebP**.
  - Генерация 3 версий каждого файла: `thumb` (квадратная миниатюра), `medium` (оптимизированное превью) и `original` (полноразмерный WebP).
  - Чтение технических **EXIF-метаданных** (камера, объектив, фокусное расстояние, диафрагма, выдержка, ISO).
  - Принудительное удаление приватных геоданных (GPS) при сохранении.
- 🔍 **Управление масштабом и плотностью сетки:**
  - Динамическое приближение и отдаление всей экспозиции (от 1 крупного кадра в ряд в стиле editorial-фотоленты до 6 колонок для панорамного обзора коллекции).
  - Сохранение масштаба и выбранной цветовой темы (Light / Dark) в `localStorage`.
- 🗂️ **Коллекции и приватность:**
  - Создание тематических альбомов.
  - Разделение прав доступа: публичные и приватные коллекции (скрыты от неавторизованных посетителей).
  - Нативная адресация: поддержка прямых ссылок на коллекции (`/album/:id`) и авторов (`/author/:username`).
- 🔐 **Безопасность корпоративного уровня:**
  - Аутентификация через **JWT** в защищённых `HttpOnly; SameSite=Lax; Secure` cookies.
  - Мгновенный отзыв токенов при выходе (Blacklist в базе данных).
  - Глобальный сброс сессий пользователя при изменении пароля администратором через версионирование токенов (`token_version`).
  - Пароли хэшируются **bcrypt (cost 12)** с защитой от атак по времени (Timing Attack mitigation).
  - Аппаратный Rate Limiting на уровне Nginx против брутфорс-атак на эндпоинты входа.
- 💾 **Гибкость хранилища:**
  - Поддержка монтирования хранилища фотографий на отдельный физический диск или сетевой раздел через переменную окружения `PHOTOS_STORAGE_PATH`.

---

## 🛠️ Стек технологий

### Frontend
- **Ядро:** [React 18](https://react.dev/), [Vite 5](https://vitejs.dev/)
- **Стилизация:** SCSS (CSS Modules), CSS Custom Properties для бесшовной смены тем (Light / Dark)
- **Иконки:** [Lucide React](https://lucide.dev/)
- **Особенности:** Zero-dependency подход (без тяжелых UI-фреймворков), нативный History API, адаптивная верстка (Mobile / Tablet / Desktop / Ultra-wide).

### Backend
- **Язык:** [Go 1.22](https://go.dev/)
- **Роутер:** [go-chi/chi v5](https://github.com/go-chi/chi) — сверхбыстрый и легковесный HTTP-маршрутизатор
- **Работа с изображениями:** [imaging](https://github.com/disintegration/imaging), [chai2010/webp](https://github.com/chai2010/webp)
- **EXIF-парсер:** [rwcarlsen/goexif](https://github.com/rwcarlsen/goexif)
- **Токены:** [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt)
- **Криптография:** [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

### База данных
- **СУБД:** [PostgreSQL 16](https://www.postgresql.org/)
- **Драйвер:** [pgx v5](https://github.com/jackc/pgx) с использованием постоянного пула соединений (`pgxpool`)
- **Миграции:** версионированные SQL-миграции с автоматическим накатыванием при старте сервиса.

### Инфраструктура и доставка
- **Docker & Docker Compose:** мультистейдж-сборка Go-бэкенда на базе легковесного Alpine Linux.
- **Nginx Alpine:** 
  - Раздача статики SPA с роутингом на `index.html`.
  - Прямой кэширующий стриминг медиафайлов из `/media/` с заголовками `Cache-Control: public, max-age=30d, immutable` в обход Go-бэкенда.
  - Проксирование API-запросов (`/api/`) с поддержкой WebSocket/HTTP-upgrades и пробросом реальных IP (`set_real_ip_from`).

---

## 📂 Структура проекта

```text
PhotoService/
├── backend/
│   ├── cmd/api/                 # Точка входа в приложение (main.go)
│   ├── internal/
│   │   ├── config/              # Загрузка и валидация переменных окружения
│   │   ├── handler/             # HTTP-контроллеры (auth, photo, album)
│   │   ├── middleware/          # JWT-сессии, проверка прав и ролей (Admin/User)
│   │   ├── model/               # Структуры данных и DTO
│   │   ├── repository/          # SQL-запросы к PostgreSQL (pgxpool)
│   │   └── service/             # Бизнес-логика (хеширование, обработка изображений)
│   ├── migrations/              # SQL-миграции схемы БД
│   └── Dockerfile               # Двухэтапная сборка бинарника Go
├── frontend/
│   ├── src/
│   │   ├── components/          # Компоненты UI (Header, PhotoGrid, Lightbox, Modals)
│   │   ├── services/            # Модуль взаимодействия с API (fetch)
│   │   └── styles/              # SCSS-переменные, типографика и глобальные стили
│   ├── package.json
│   └── vite.config.js
├── nginx/
│   ├── default.conf             # Конфигурация виртуального хоста Nginx (SPA + Media + API)
│   └── nginx.conf               # Глобальные параметры Nginx, gzip и зоны rate-limit
├── docker-compose.yml           # Декларация контейнеров (Postgres, Backend, Nginx)
├── .env.example                 # Шаблон переменных окружения
└── README.md
```

---

## 🚀 Быстрый старт (Локальная разработка)

### Предварительные требования
- Установленный [Docker](https://docs.docker.com/get-docker/) и [Docker Compose](https://docs.docker.com/compose/)
*(Node.js на сервере не требуется — фронтенд и бэкенд компилируются внутри Docker-контейнеров).*

### 1. Клонирование и настройка окружения
```bash
git clone <url-вашего-репозитория> PhotoService
cd PhotoService

# Создаём локальный .env файл
cp .env.example .env
```

### 2. Запуск контейнеров
```bash
docker compose up --build -d
```

После завершения сборки галерея доступна в браузере по адресу:  
👉 **`http://localhost`**

---

## ⚙️ Конфигурация (`.env`)

| Переменная | По умолчанию | Описание |
| :--- | :--- | :--- |
| `POSTGRES_DB` | `photogallery` | Имя базы данных PostgreSQL |
| `POSTGRES_USER` | `photouser` | Пользователь базы данных |
| `POSTGRES_PASSWORD` | `photosecret` | Пароль базы данных (обязательно смените на проде!) |
| `DATABASE_URL` | `postgres://...` | DSN-строка подключения к БД |
| `PORT` | `8080` | Внутренний порт Go API |
| `APP_PORT` | `80` (локально) / `8080` (прод) | Порт Nginx, пробрасываемый на хост |
| `APP_BIND_IP` | `0.0.0.0` (локально) / `127.0.0.1` | Сетевой интерфейс привязки портов |
| `JWT_SECRET` | `photogallery_...` | Секретный ключ подписи JWT (от 32 символов) |
| `COOKIE_SECURE` | `false` (локально) / `true` (HTTPS) | Флаг Secure для сессионных кук |
| `UPLOAD_DIR` | `/uploads` | Внутренний каталог хранения снимков в контейнере |
| `MAX_UPLOAD_SIZE` | `20971520` | Максимальный размер загружаемого файла (20 МБ) |
| `PHOTOS_STORAGE_PATH` | `./uploads` | **Путь на хосте/диске** для хранения файлов |
| `CORS_ALLOWED_ORIGINS`| `""` | Список доверенных доменов через запятую (для HTTPS) |

---

## 🌐 Развёртывание на продакшене (HTTPS & Поддомен)

Для запуска на сервере под отдельным поддоменом (например, `photos.yourdomain.com`) за внешним хостовым Nginx с SSL-сертификатом от Let's Encrypt:

### 1. Настройка `.env` на сервере
```env
APP_BIND_IP=127.0.0.1
APP_PORT=8080
COOKIE_SECURE=true
CORS_ALLOWED_ORIGINS=https://photos.yourdomain.com
PHOTOS_STORAGE_PATH=/mnt/storage/photos
JWT_SECRET=super_secret_jwt_random_string_32_chars!
POSTGRES_PASSWORD=your_strong_postgres_password
DATABASE_URL=postgres://photouser:your_strong_postgres_password@postgres:5432/photogallery?sslmode=disable
```

### 2. Хранение фото на отдельном диске
Если снимки должны лежать на отдельном смонтированном диске:
```bash
sudo mkdir -p /mnt/storage/photos
sudo chmod 777 /mnt/storage/photos
```

### 3. Конфигурация хостового Nginx (`/etc/nginx/sites-available/photos.yourdomain.com`)
```nginx
server {
    listen 80;
    server_name photos.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name photos.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/photos.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/photos.yourdomain.com/privkey.pem;

    client_max_body_size 25M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 90s;
    }
}
```

---

## 👑 Панель управления и роли

В системе предусмотрена двухуровневая модель доступа:

1. **Администратор (Куратор):**
   - Управление авторами (создание, удаление пользователей, сброс паролей).
   - Управление любыми альбомами и переключение их публичности.
   - Удаление любых неподобающих фотографий.
   - Доступ к скрытым/приватным снимкам и коллекциям.
2. **Автор (User):**
   - Загрузка авторских фотографий с автоматическим извлечением параметров съемки.
   - Создание персональных коллекций и управление доступом к ним.
   - Редактирование и удаление собственных работ.

---

## 📄 Лицензия

Проект распространяется под лицензией **MIT**. Подробности в файле `LICENSE` (при наличии).

# 📷 PhotoGallery (PhotoService)

<p align="center">
  <b>Минималистичная, высокопроизводительная веб-галерея для авторской фотографии</b><br>
  Фокус на эстетике, скорости доставки контента, строгой защите приватности и чистоте визуального восприятия.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-1.3.0-black?style=for-the-badge" alt="Version 1.3.0">
  <img src="https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.22">
  <img src="https://img.shields.io/badge/React-18.3-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React 18">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL 16">
  <img src="https://img.shields.io/badge/Nginx-Latest-009639?style=for-the-badge&logo=nginx&logoColor=white" alt="Nginx">
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
  - Автоматическая инициализация и синхронизация учетной записи администратора из `.env` (`ADMIN_USERNAME` / `ADMIN_PASSWORD`).
  - Глобальный сброс сессий пользователя при изменении пароля администратором через версионирование токенов (`token_version`).
  - Пароли хэшируются **bcrypt (cost 12)** с защитой от атак по времени (Timing Attack mitigation).
  - Аппаратный Rate Limiting на уровне Nginx против брутфорс-атак на эндпоинты входа.
  - Чистая база без тестовых/демонстрационных учетных записей.
- 💾 **Гибкость хранилища:**
  - Поддержка монтирования хранилища фотографий на отдельный физический диск или сетевой раздел через переменную окружения `PHOTOS_STORAGE_PATH`.

---

## 🛠️ Стек технологий

### Frontend
- **Ядро:** [React 18](https://react.dev/), [Vite 5](https://vitejs.dev/)
- **Стилизация:** SCSS (CSS Modules), CSS Custom Properties для бесшовной смены тем (Light / Dark)
- **Иконки:** [Lucide React](https://lucide.dev/)
- **Особенности:** Zero-dependency подход, адаптивная верстка (Mobile / Tablet / Desktop / Ultra-wide).
- **Сборка:** Скомпилированный продакшн-билд (`frontend/dist`) хранится в репозитории — на сервере **не требуются** Node.js и npm.

### Backend
- **Язык:** [Go 1.22](https://go.dev/)
- **Роутер:** [go-chi/chi v5](https://github.com/go-chi/chi) — быстрый и легковесный HTTP-маршрутизатор
- **Работа с изображениями:** [imaging](https://github.com/disintegration/imaging), [chai2010/webp](https://github.com/chai2010/webp)
- **EXIF-парсер:** [rwcarlsen/goexif](https://github.com/rwcarlsen/goexif)
- **Токены:** [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt)
- **Криптография:** [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

### База данных
- **СУБД:** [PostgreSQL 16](https://www.postgresql.org/)
- **Драйвер:** [pgx v5](https://github.com/jackc/pgx) с пулом соединений (`pgxpool`)
- **Миграции:** версионированные SQL-миграции с автоматическим накатыванием и очисткой устаревших демо-данных.

### Инфраструктура и доставка
- **Docker Compose:** изолированная среда из трёх сервисов:
  - `photo_postgres` — база данных PostgreSQL 16 Alpine с постоянным volume `postgres_data`.
  - `photo_backend` — скомпилированный Go REST API микросервис.
  - `photo_nginx` — официальный образ `nginx:latest`, отдающий статику SPA, медиафайлы и проксирующий `/api/`.
- **Nginx архитектура:**
  - Внутренний DNS-резолвер (`resolver 127.0.0.11`) с динамическим апстримом предотвращает падение Nginx при старте или перезапуске бэкенда.
  - Прямой кэширующий стриминг медиафайлов из `/media/` с заголовками `Cache-Control: public, max-age=30d, immutable` в обход Go-бэкенда.
  - Корректная поддержка SPA-роутинга (`try_files $uri $uri/ /index.html`).

---

## 📂 Структура проекта

```text
PhotoService/
├── backend/
│   ├── cmd/api/                 # Точка входа в приложение (main.go)
│   ├── internal/
│   │   ├── config/              # Загрузка и валидация конфигурации (.env)
│   │   ├── handler/             # HTTP-контроллеры (auth, photo, album)
│   │   ├── middleware/          # JWT-сессии, проверка прав и ролей (Admin/User)
│   │   ├── model/               # Структуры данных и DTO
│   │   ├── repository/          # SQL-запросы к PostgreSQL (pgxpool)
│   │   └── service/             # Бизнес-логика (хеширование, обработка изображений)
│   ├── migrations/              # SQL-миграции схемы БД
│   └── Dockerfile               # Мультистейдж-сборка Go-бинарника
├── frontend/
│   ├── dist/                    # Скомпилированный production билд (монтируется в Nginx)
│   ├── src/                     # Исходный код React-приложения
│   ├── package.json
│   └── vite.config.js
├── nginx/
│   ├── default.conf             # Конфигурация виртуального хоста (SPA + Media + API)
│   └── nginx.conf               # Глобальные параметры Nginx, gzip и зоны rate-limit
├── docker-compose.yml           # Декларация сервисов (postgres, backend, photo_nginx)
├── .env.example                 # Шаблон переменных окружения
└── README.md
```

---

## ⚙️ Конфигурация (`.env`)

| Переменная | По умолчанию | Описание |
| :--- | :--- | :--- |
| `POSTGRES_DB` | `photogallery` | Имя базы данных PostgreSQL |
| `POSTGRES_USER` | `photouser` | Пользователь базы данных |
| `POSTGRES_PASSWORD` | `photosecret` | Пароль базы данных (обязательно задайте надёжный!) |
| `DATABASE_URL` | `postgres://...` | DSN-строка подключения к PostgreSQL |
| `PORT` | `8080` | Внутренний порт Go API |
| `APP_PORT` | `80` (локально) / `8080` (прод) | Порт Nginx, пробрасываемый на хост |
| `APP_BIND_IP` | `0.0.0.0` (локально) / `127.0.0.1` | Сетевой интерфейс привязки портов |
| `ADMIN_USERNAME` | `admin` | Логин начального администратора |
| `ADMIN_PASSWORD` | `admin123` | Пароль администратора (синхронизируется с БД при старте) |
| `JWT_SECRET` | `photogallery_...` | Секретный ключ подписи JWT (от 32 символов) |
| `COOKIE_SECURE` | `false` (локально) / `true` (HTTPS) | Флаг Secure для сессионных кук |
| `UPLOAD_DIR` | `/uploads` | Внутренний каталог хранения снимков в контейнере |
| `MAX_UPLOAD_SIZE` | `20971520` | Максимальный размер загружаемого файла (20 МБ) |
| `PHOTOS_STORAGE_PATH` | `./uploads` | **Путь на хосте/диске** для хранения файлов фото |
| `CORS_ALLOWED_ORIGINS`| `""` | Список доверенных доменов через запятую (для HTTPS) |

---

## 🚀 Развёртывание на продакшене (HTTPS & Поддомен)

Схема работы: Внешний Nginx сервера принимает HTTPS трафик с SSL-сертификатом Let's Encrypt и проксирует его на локальный порт контейнера `127.0.0.1:8080` (`photo_nginx`).

### 1. Подготовка окружения на сервере
Клонируйте репозиторий и создайте рабочий `.env`:
```bash
git clone https://github.com/LiveIsAbsurd/PhotoAbsurd.git PhotoService
cd PhotoService
cp .env.example .env
nano .env
```

Заполните ваши боевые настройки:
```env
APP_BIND_IP=127.0.0.1
APP_PORT=8080
COOKIE_SECURE=true
CORS_ALLOWED_ORIGINS=https://photo.yourdomain.com
PHOTOS_STORAGE_PATH=/mnt/storage/photos
JWT_SECRET=сгенерируйте_случайную_строку_от_32_символов
POSTGRES_PASSWORD=сложный_пароль_базы_данных
DATABASE_URL=postgres://photouser:сложный_пароль_базы_данных@postgres:5432/photogallery?sslmode=disable
ADMIN_USERNAME=ваш_логин_админа
ADMIN_PASSWORD=ваш_надежный_пароль_админа
```

### 2. Права на каталог хранилища
Если снимки хранятся на отдельном диске или в отдельной папке:
```bash
sudo mkdir -p /mnt/storage/photos
sudo chmod 777 /mnt/storage/photos
```

### 3. Конфигурация внешнего Nginx на хосте (`/etc/nginx/sites-available/photo.yourdomain.com`)
```nginx
server {
    listen 80;
    server_name photo.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name photo.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/photo.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/photo.yourdomain.com/privkey.pem;

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

Активируйте конфиг и перезапустите системный Nginx:
```bash
sudo ln -s /etc/nginx/sites-available/photo.yourdomain.com /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### 4. Запуск контейнеров
```bash
sudo docker compose up --build -d
```

---

## 🔄 Обновление проекта на сервере

Благодаря тому, что фронтенд компилируется заранее, а Nginx использует официальный образ:
```bash
# 1. Загружаем свежие коммиты
git pull

# 2. Пересобираем бэкенд и перезапускаем сервисы
sudo docker compose up --build -d
```

---

## 🛠️ Полезные команды обслуживания

- **Проверить логи любого сервиса:**
  ```bash
  sudo docker compose logs -f backend
  sudo docker compose logs -f photo_nginx
  ```
- **Посмотреть пользователей в базе данных:**
  ```bash
  sudo docker compose exec postgres psql -U photouser -d photogallery -c "SELECT id, username, role, created_at FROM users;"
  ```
- **Сбросить пароль администратора:**
  Измените `ADMIN_PASSWORD` в файле `.env` и примените изменения:
  ```bash
  sudo docker compose up -d backend
  ```
  *(Бэкенд автоматически проверит и обновит хеш пароля в базе данных при старте).*

---

## 👑 Панель управления и роли

1. **Администратор (Куратор):**
   - Управление авторами (создание новых пользователей, сброс паролей).
   - Управление альбомами и их видимостью.
   - Загрузка, редактирование и удаление любых снимков.
   - Просмотр приватных коллекций.
2. **Автор (User):**
   - Загрузка авторских фотографий с автоматическим извлечением параметров съемки.
   - Создание персональных коллекций.
   - Редактирование и удаление собственных работ.

---

## 📄 Лицензия

Проект распространяется под лицензией **MIT**.

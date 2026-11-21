package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"photogallery/internal/config"
	"photogallery/internal/handler"
	"photogallery/internal/middleware"
	"photogallery/internal/repository"
	"photogallery/internal/service"
)

// main — главная точка входа в приложение на языке Go.
// Здесь происходит инициализация конфигурации, подключение к базе данных,
// настройка маршрутизатора (роутера), регистрация middleware и запуск HTTP-сервера.
func main() {
	// 1. Загрузка настроек приложения из переменных окружения
	appConfig := config.Load()

	// 2. Создание корневого контекста с возможностью отмены (Graceful Shutdown)
	rootContext, cancelRootContext := context.WithCancel(context.Background())
	defer cancelRootContext() // defer гарантирует вызов функции при выходе из main

	// 3. Подключение к пулу соединений базы данных PostgreSQL
	log.Println("Подключение к базе данных PostgreSQL...")
	database, err := repository.NewDB(rootContext, appConfig.DatabaseURL)
	if err != nil {
		log.Fatalf("Критическая ошибка подключения к базе данных: %v", err)
	}
	defer database.Close() // Закрываем пул соединений при завершении программы
	log.Println("Успешное подключение к PostgreSQL.")

	// 4. Инициализация сервисного слоя (бизнес-логика)
	// Сервис обработки изображений (ресайз, конвертация в WebP, очистка EXIF)
	imageService := service.NewImageService(appConfig.UploadDir)
	// Сервис аутентификации (хеширование паролей bcrypt, выпуск и проверка JWT-токенов)
	authService := service.NewAuthService(appConfig.JWTSecret)

	// 5. Инициализация репозиториев (прямая работа с таблицами БД)
	userRepository := repository.NewUserRepository(database)
	photoRepository := repository.NewPhotoRepository(database)
	albumRepository := repository.NewAlbumRepository(database)
	likeRepository := repository.NewLikeRepository(database)

	// 6. Инициализация промежуточных обработчиков (Middleware)
	appMiddleware := middleware.NewMiddleware(authService, userRepository, appConfig.CookieSecure)

	// 7. Инициализация HTTP-контроллеров (обработчиков запросов)
	authHandler := handler.NewAuthHandler(userRepository, photoRepository, authService, appConfig.UploadDir, appConfig.CookieSecure)
	photoHandler := handler.NewPhotoHandler(photoRepository, albumRepository, likeRepository, imageService, appConfig.UploadDir)
	albumHandler := handler.NewAlbumHandler(albumRepository, photoRepository)

	// 8. Настройка маршрутизатора Chi Router
	router := chi.NewRouter()

	// Стандартные middleware для обеспечения надёжности и мониторинга:
	router.Use(chimiddleware.RequestID) // Присваивает каждому запросу уникальный ID
	router.Use(chimiddleware.RealIP)    // Извлекает реальный IP клиента из заголовков X-Forwarded-For
	router.Use(chimiddleware.Logger)    // Логирует каждый входящий запрос (метод, путь, статус, время)
	router.Use(chimiddleware.Recoverer) // Перехватывает паники (panic) и возвращает 500 ошибку без падения сервера
	router.Use(chimiddleware.Timeout(60 * time.Second)) // Ограничение на выполнение запроса

	// Настройка CORS (Cross-Origin Resource Sharing) для безопасной работы с браузером
	corsOrigins := []string{
		"http://localhost",
		"http://localhost:80",
		"http://localhost:5173",
		"http://localhost:3000",
		"http://127.0.0.1",
		"http://127.0.0.1:80",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
	}
	if len(appConfig.CORSAllowedOrigins) > 0 {
		corsOrigins = append(corsOrigins, appConfig.CORSAllowedOrigins...)
	}

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Разрешаем передачу HttpOnly cookies
		MaxAge:           300,
	}))

	// Подключаем наш middleware сессий: гостевой токен и распаковка JWT пользователя
	router.Use(appMiddleware.SessionMiddleware)

	// 9. Регистрация маршрутов API
	router.Route("/api", func(apiRouter chi.Router) {
		// Проверка здоровья сервиса (Health Check)
		apiRouter.Get("/health", func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusOK)
			_, _ = responseWriter.Write([]byte(`{"status":"ok"}`))
		})

		// Маршруты авторизации
		apiRouter.Route("/auth", func(authRouter chi.Router) {
			authRouter.Post("/login", authHandler.Login)   // Вход по логину и паролю
			authRouter.Post("/logout", authHandler.Logout) // Выход с очисткой куки
			authRouter.Get("/me", authHandler.Me)          // Данные текущего пользователя
		})

		// Маршруты панели администратора (доступ только для role = admin)
		apiRouter.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(appMiddleware.RequireAuth)  // Проверка: пользователь авторизован
			adminRouter.Use(appMiddleware.RequireAdmin) // Проверка: роль пользователя — admin

			adminRouter.Get("/users", authHandler.AdminListUsers)                 // Список пользователей
			adminRouter.Post("/users", authHandler.AdminCreateUser)               // Создание нового пользователя
			adminRouter.Put("/users/{id}/password", authHandler.AdminChangeUserPassword) // Смена пароля пользователя админом
			adminRouter.Delete("/users/{id}", authHandler.AdminDeleteUser)        // Удаление пользователя
		})

		// Маршруты для работы с фотографиями
		apiRouter.Route("/photos", func(photoRouter chi.Router) {
			photoRouter.Get("/", photoHandler.List)                 // Публичная лента фотографий
			photoRouter.Post("/", photoHandler.Upload)              // Загрузка фото (только для авторизованных)
			photoRouter.Get("/{id}", photoHandler.GetByID)          // Детальная информация о фото
			photoRouter.Patch("/{id}", photoHandler.Update)         // Редактирование названия/описания
			photoRouter.Delete("/{id}", photoHandler.Delete)        // Удаление (автор своего фото или админ)
		})

		// Маршруты для работы с альбомами (коллекциями)
		apiRouter.Route("/albums", func(albumRouter chi.Router) {
			albumRouter.Get("/", albumHandler.List)          // Список альбомов
			albumRouter.Post("/", albumHandler.Create)       // Создание нового альбома
			albumRouter.Get("/{id}", albumHandler.GetByID)   // Просмотр альбома со снимками
			albumRouter.Patch("/{id}", albumHandler.Update)  // Обновление настроек и доступа к альбому
			albumRouter.Delete("/{id}", albumHandler.Delete) // Удаление альбома
		})

		// Пользовательские разделы
		apiRouter.Route("/users", func(userRouter chi.Router) {
			userRouter.Group(func(protectedUserRouter chi.Router) {
				protectedUserRouter.Use(appMiddleware.RequireAuth)
				protectedUserRouter.Get("/me/photos", photoHandler.ListUserPhotos) // Загрузки пользователя
			})
		})
	})

	// 10. Конфигурация HTTP-сервера с защитными таймаутами
	httpServer := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,  // Максимальное время чтения тела запроса
		WriteTimeout: 60 * time.Second,  // Максимальное время записи ответа клиенту
		IdleTimeout:  120 * time.Second, // Время удержания открытого неактивного соединения
	}

	// 11. Запуск сервера в отдельной горутине (фоновом потоке Go)
	go func() {
		log.Printf("Сервер PhotoService запущен и слушает порт %s...", appConfig.Port)
		if serverError := httpServer.ListenAndServe(); serverError != nil && serverError != http.ErrServerClosed {
			log.Fatalf("Ошибка работы HTTP-сервера: %v", serverError)
		}
	}()

	// 12. Ожидание системного сигнала прерывания (Ctrl+C или docker stop) для плавной остановки
	quitSignalChannel := make(chan os.Signal, 1)
	signal.Notify(quitSignalChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitSignalChannel // Блокировка main до получения сигнала остановки
	log.Println("Получен сигнал завершения. Плавная остановка сервера (Graceful Shutdown)...")

	// Даем 10 секунд на корректное завершение всех активных запросов
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownContext); err != nil {
		log.Fatalf("Сервер был принудительно остановлен по ошибке: %v", err)
	}

	log.Println("Сервер успешно остановлен. Все ресурсы освобождены.")
}

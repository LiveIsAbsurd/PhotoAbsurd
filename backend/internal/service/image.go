package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // Импорт с подчёркиванием регистрирует GIF-декодер в стандартном пакете image
	_ "image/jpeg" // Регистрирует JPEG-декодер
	_ "image/png"  // Регистрирует PNG-декодер
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"

	"photogallery/internal/model"
)

// init вызывается Go автоматически при старте программы до выполнения main()
func init() {
	// Регистрируем парсеры заметок производителей камер (Nikon, Canon, Sony и др.)
	exif.RegisterParsers(mknote.All...)
}

// Ошибки конвейера валидации и обработки изображений
var (
	ErrInvalidImageType    = errors.New("недопустимый формат: разрешены только JPEG, PNG и WebP")
	ErrImageDimensionsBomb = errors.New("разрешение изображения превышает безопасный предел 8192x8192 пикселей")
)

// ProcessedImageResult — результат работы конвейера обработки изображения
type ProcessedImageResult struct {
	ID               uuid.UUID        // Уникальный идентификатор файла
	FilePathOriginal string           // Относительный веб-путь к оптимизированному оригиналу
	FilePathMedium   string           // Относительный веб-путь к среднему размеру (1600px)
	FilePathThumb    string           // Относительный веб-путь к квадратной миниатюре (400px)
	Width            int              // Ширина исходного изображения
	Height           int              // Высота исходного изображения
	FileSize         int64            // Размер загруженного файла в байтах
	CameraMeta       model.CameraMeta // Извлечённые метаданные EXIF
}

// ImageService отвечает за валидацию, очистку EXIF, нарезку миниатюр и сохранение фото
type ImageService struct {
	uploadDirectoryPath string
}

// NewImageService — конструктор сервиса обработки изображений
func NewImageService(uploadDirectoryPath string) *ImageService {
	// Создаем каталог на диске, если он ещё не существует
	_ = os.MkdirAll(uploadDirectoryPath, 0755)
	return &ImageService{uploadDirectoryPath: uploadDirectoryPath}
}

// ProcessAndSave принимает поток байтов загруженного файла и выполняет полный цикл обработки:
// 1. Проверяет сигнатуру (Magic Bytes), отбрасывая небезопасные файлы
// 2. Извлекает технический EXIF (камера, выдержка, ISO), вырезая приватный GPS
// 3. Проверяет габариты через DecodeConfig БЕЗ полной распаковки в память (защита от Pixel Flood)
// 4. Поворачивает фото согласно ориентации EXIF (AutoOrientation)
// 5. Генерирует 3 WebP-файла: оригинал, medium (1600px), thumb (400px)
func (service *ImageService) ProcessAndSave(uploadStream io.Reader) (*ProcessedImageResult, error) {
	// Считываем данные из потока в срез байтов []byte
	fileDataBytes, readError := io.ReadAll(uploadStream)
	if readError != nil {
		return nil, fmt.Errorf("ошибка чтения загруженного файла: %w", readError)
	}

	if len(fileDataBytes) < 512 {
		return nil, errors.New("файл слишком мал, чтобы быть корректным изображением")
	}

	// 1. Проверка Magic Bytes: определяем реальный MIME-тип по первым 512 байтам содержимого,
	// а не по расширению файла (защита от поддельных файлов)
	detectedMimeType := http.DetectContentType(fileDataBytes[:512])
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowedMimeTypes[detectedMimeType] {
		return nil, ErrInvalidImageType
	}

	// 2. Извлекаем параметры камеры из метаданных EXIF (до перекодирования)
	cameraMetadata := extractCameraEXIF(fileDataBytes)

	// 3. Защита от пиксельных бомб (Decompression Bomb):
	// DecodeConfig считывает только заголовок с размерами без выделения оперативной памяти под миллионы пикселей
	imageConfig, _, configError := image.DecodeConfig(bytes.NewReader(fileDataBytes))
	if configError != nil {
		return nil, fmt.Errorf("не удалось прочитать размеры изображения: %w", configError)
	}
	if imageConfig.Width > 8192 || imageConfig.Height > 8192 {
		return nil, ErrImageDimensionsBomb
	}

	// 4. Полное декодирование растра с автоповоротом по тегу ориентации EXIF
	decodedImage, decodeError := imaging.Decode(bytes.NewReader(fileDataBytes), imaging.AutoOrientation(true))
	if decodeError != nil {
		return nil, fmt.Errorf("ошибка декодирования изображения: %w", decodeError)
	}

	imageBounds := decodedImage.Bounds()
	originalWidth := imageBounds.Dx()
	originalHeight := imageBounds.Dy()
	fileUUID := uuid.New()

	// 5А. Создание оптимизированного оригинала (уменьшаем до 3840px по ширине, если снимок гигантский)
	optimizedOriginalImage := decodedImage
	if originalWidth > 3840 {
		optimizedOriginalImage = imaging.Resize(decodedImage, 3840, 0, imaging.Lanczos)
	}
	originalFileName := fmt.Sprintf("%s_original.webp", fileUUID)
	originalDiskPath := filepath.Join(service.uploadDirectoryPath, originalFileName)
	if writeError := saveImageAsWebP(originalDiskPath, optimizedOriginalImage, 90); writeError != nil {
		return nil, fmt.Errorf("не удалось сохранить оригинал в формате WebP: %w", writeError)
	}

	// 5Б. Создание версии Medium (вписываем в квадрат 1600x1600 с сохранением пропорций)
	mediumImage := imaging.Fit(decodedImage, 1600, 1600, imaging.Lanczos)
	mediumFileName := fmt.Sprintf("%s_medium.webp", fileUUID)
	mediumDiskPath := filepath.Join(service.uploadDirectoryPath, mediumFileName)
	if writeError := saveImageAsWebP(mediumDiskPath, mediumImage, 85); writeError != nil {
		return nil, fmt.Errorf("не удалось сохранить среднюю версию WebP: %w", writeError)
	}

	// 5В. Создание квадратной миниатюры Thumb (400x400 с центрированной обрезкой)
	thumbnailImage := imaging.Fill(decodedImage, 400, 400, imaging.Center, imaging.Lanczos)
	thumbnailFileName := fmt.Sprintf("%s_thumb.webp", fileUUID)
	thumbnailDiskPath := filepath.Join(service.uploadDirectoryPath, thumbnailFileName)
	if writeError := saveImageAsWebP(thumbnailDiskPath, thumbnailImage, 80); writeError != nil {
		return nil, fmt.Errorf("не удалось сохранить миниатюру WebP: %w", writeError)
	}

	return &ProcessedImageResult{
		ID:               fileUUID,
		FilePathOriginal: "/media/" + originalFileName,
		FilePathMedium:   "/media/" + mediumFileName,
		FilePathThumb:    "/media/" + thumbnailFileName,
		Width:            originalWidth,
		Height:           originalHeight,
		FileSize:         int64(len(fileDataBytes)),
		CameraMeta:       cameraMetadata,
	}, nil
}

// saveImageAsWebP кодирует изображение в формат WebP с заданным качеством и записывает в файл
func saveImageAsWebP(destinationFilePath string, sourceImage image.Image, qualityScore float32) error {
	destinationFile, createError := os.Create(destinationFilePath)
	if createError != nil {
		return createError
	}
	defer destinationFile.Close()

	// Перекодирование в WebP полностью удаляет любые скрытые вредоносные бинарные вставки
	return webp.Encode(destinationFile, sourceImage, &webp.Options{
		Lossless: false,
		Quality:  qualityScore,
	})
}

// extractCameraEXIF считывает технические параметры съёмки (диафрагма, выдержка, ISO, фокусное расстояние).
// Данные GPS намеренно игнорируются для сохранения конфиденциальности автора снимка.
// Функция защищена recover(), чтобы повреждённый или вредоносный EXIF не приводил к панике сервера.
func extractCameraEXIF(imageBytes []byte) (extractedMetadata model.CameraMeta) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			// В случае паники в парсере сторонней библиотеки goexif возвращаем пустые метаданные
			extractedMetadata = model.CameraMeta{}
		}
	}()

	exifData, parseError := exif.Decode(bytes.NewReader(imageBytes))
	if parseError != nil {
		return extractedMetadata // EXIF отсутствует или повреждён
	}

	if makeTag, err := exifData.Get(exif.Make); err == nil {
		extractedMetadata.Make = cleanMetadataString(makeTag.String())
	}
	if modelTag, err := exifData.Get(exif.Model); err == nil {
		extractedMetadata.Model = cleanMetadataString(modelTag.String())
	}
	if lensTag, err := exifData.Get(exif.LensModel); err == nil {
		extractedMetadata.Lens = cleanMetadataString(lensTag.String())
	}
	if focalTag, err := exifData.Get(exif.FocalLength); err == nil {
		if numerator, denominator, ratErr := focalTag.Rat2(0); ratErr == nil && denominator != 0 {
			extractedMetadata.FocalLength = fmt.Sprintf("%.0f mm", float64(numerator)/float64(denominator))
		}
	}
	if fNumberTag, err := exifData.Get(exif.FNumber); err == nil {
		if numerator, denominator, ratErr := fNumberTag.Rat2(0); ratErr == nil && denominator != 0 {
			extractedMetadata.FNumber = fmt.Sprintf("f/%.1f", float64(numerator)/float64(denominator))
		}
	}
	if exposureTag, err := exifData.Get(exif.ExposureTime); err == nil {
		if numerator, denominator, ratErr := exposureTag.Rat2(0); ratErr == nil && denominator != 0 {
			if numerator == 1 {
				extractedMetadata.ExposureTime = fmt.Sprintf("1/%d s", denominator)
			} else {
				extractedMetadata.ExposureTime = fmt.Sprintf("%.2f s", float64(numerator)/float64(denominator))
			}
		}
	}
	if isoTag, err := exifData.Get(exif.ISOSpeedRatings); err == nil {
		extractedMetadata.ISO = fmt.Sprintf("ISO %s", cleanMetadataString(isoTag.String()))
	}
	if captureTime, err := exifData.DateTime(); err == nil {
		extractedMetadata.DateTime = captureTime.Format("2006-01-02 15:04:05")
	}

	return extractedMetadata
}

// cleanMetadataString очищает строки метаданных от кавычек и служебных нуль-байтов
func cleanMetadataString(rawString string) string {
	cleaned := strings.Trim(rawString, "\" \t\n\r")
	return strings.ReplaceAll(cleaned, "\x00", "")
}

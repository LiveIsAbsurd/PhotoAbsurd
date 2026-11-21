import React, { useEffect } from 'react';
import { X, ChevronLeft, ChevronRight, Camera, Trash2 } from 'lucide-react';
import styles from './Lightbox.module.scss';

/**
 * Lightbox — полноэкранный модальный просмотр фотографии в оригинальном разрешении.
 * Поддерживает:
 * - Навигацию стрелками на клавиатуре (Escape, Влево, Вправо)
 * - Отображение технических параметров съемки (EXIF)
 * - Удаление снимка автором или администратором
 */
export default function Lightbox({
  currentPhoto,
  onCloseLightbox,
  onNavigateToPreviousPhoto,
  onNavigateToNextPhoto,
  hasPreviousPhoto,
  hasNextPhoto,
  onDeletePhoto,
  currentUser,
  onSelectAuthor,
}) {
  // Обработка нажатий клавиш на клавиатуре (Keyboard Navigation)
  useEffect(() => {
    const handleKeyboardKeyDown = (keyboardEvent) => {
      if (keyboardEvent.key === 'Escape') {
        onCloseLightbox();
      }
      if (keyboardEvent.key === 'ArrowLeft' && hasPreviousPhoto) {
        onNavigateToPreviousPhoto();
      }
      if (keyboardEvent.key === 'ArrowRight' && hasNextPhoto) {
        onNavigateToNextPhoto();
      }
    };

    window.addEventListener('keydown', handleKeyboardKeyDown);
    // Блокируем прокрутку основной страницы, пока открыт Lightbox
    document.body.style.overflow = 'hidden';

    return () => {
      window.removeEventListener('keydown', handleKeyboardKeyDown);
      document.body.style.overflow = '';
    };
  }, [onCloseLightbox, onNavigateToPreviousPhoto, onNavigateToNextPhoto, hasPreviousPhoto, hasNextPhoto]);

  if (!currentPhoto) return null;

  const cameraMetadata = currentPhoto.camera_meta;
  const hasValidExifData =
    cameraMetadata &&
    (cameraMetadata.make ||
      cameraMetadata.model ||
      cameraMetadata.iso ||
      cameraMetadata.f_number ||
      cameraMetadata.focal_length);

  const canDeletePhoto =
    currentUser && (currentUser.role === 'admin' || currentUser.id === currentPhoto.user_id);

  const handleDeleteClick = () => {
    if (window.confirm('Вы действительно хотите удалить эту фотографию?')) {
      onDeletePhoto(currentPhoto.id);
    }
  };

  return (
    <div className={styles.backdrop} onClick={onCloseLightbox}>
      {/* Верхняя информационная полоса */}
      <header className={styles.topBar} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <span className={styles.counter}>{currentPhoto.title || 'БЕЗ НАЗВАНИЯ'}</span>
        <div className={styles.topActions}>
          {canDeletePhoto && (
            <button
              className={styles.deleteBtn}
              onClick={handleDeleteClick}
              title="Удалить фотографию"
            >
              <Trash2 size={16} strokeWidth={1.5} />
              <span className={styles.deleteText}>УДАЛИТЬ</span>
            </button>
          )}
          <button className={styles.closeBtn} onClick={onCloseLightbox} aria-label="Закрыть">
            <span className={styles.closeText}>ЗАКРЫТЬ</span>
            <X size={18} strokeWidth={1.5} />
          </button>
        </div>
      </header>

      {/* Центральная область со снимком и кнопками перелистывания */}
      <div className={styles.content}>
        {hasPreviousPhoto && (
          <button
            className={`${styles.navBtn} ${styles.prevBtn}`}
            onClick={(clickEvent) => {
              clickEvent.stopPropagation();
              onNavigateToPreviousPhoto();
            }}
            aria-label="Предыдущее фото"
          >
            <ChevronLeft size={28} strokeWidth={1.2} />
          </button>
        )}

        <div className={styles.imageContainer}>
          <img
            src={currentPhoto.file_path_original || currentPhoto.file_path_medium}
            alt={currentPhoto.title || 'Фотография'}
            className={styles.fullImage}
            onClick={(clickEvent) => clickEvent.stopPropagation()}
          />
        </div>

        {hasNextPhoto && (
          <button
            className={`${styles.navBtn} ${styles.nextBtn}`}
            onClick={(clickEvent) => {
              clickEvent.stopPropagation();
              onNavigateToNextPhoto();
            }}
            aria-label="Следующее фото"
          >
            <ChevronRight size={28} strokeWidth={1.2} />
          </button>
        )}
      </div>

      {/* Нижняя полоса с автором и параметрами камеры */}
      <footer className={styles.bottomBar} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <div className={styles.metaInfo}>
          <div className={styles.authorSection}>
            <span
              className={styles.authorName}
              onClick={(clickEvent) => {
                if (onSelectAuthor && currentPhoto.author_username) {
                  clickEvent.stopPropagation();
                  onSelectAuthor({
                    id: currentPhoto.user_id,
                    username: currentPhoto.author_username,
                  });
                  onCloseLightbox();
                }
              }}
              style={{ cursor: onSelectAuthor ? 'pointer' : 'default' }}
              title={currentPhoto.author_username ? `Посмотреть все работы @${currentPhoto.author_username}` : ''}
            >
              @{currentPhoto.author_username || 'автор неизвестен'}
            </span>
          </div>

          {hasValidExifData && (
            <div className={styles.exifDetails}>
              <Camera size={13} strokeWidth={1.5} className={styles.cameraIcon} />
              <span>
                {[cameraMetadata.make, cameraMetadata.model].filter(Boolean).join(' ')}
                {cameraMetadata.focal_length ? ` · ${cameraMetadata.focal_length}` : ''}
                {cameraMetadata.f_number ? ` · ${cameraMetadata.f_number}` : ''}
                {cameraMetadata.exposure_time ? ` · ${cameraMetadata.exposure_time}` : ''}
                {cameraMetadata.iso ? ` · ${cameraMetadata.iso}` : ''}
              </span>
            </div>
          )}
        </div>
      </footer>
    </div>
  );
}

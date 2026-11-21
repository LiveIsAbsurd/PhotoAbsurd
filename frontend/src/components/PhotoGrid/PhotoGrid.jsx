import React from 'react';
import PhotoCard from '../PhotoCard/PhotoCard';
import styles from './PhotoGrid.module.scss';

/**
 * Модульная сетка для отображения коллекции фотографий.
 * Если фотографий ещё нет, показывает эстетичную минималистичную заглушку.
 */
export default function PhotoGrid({
  photosList,
  isLoadingPhotos,
  onSelectPhoto,
  onOpenUploadModal,
  onDeletePhoto,
  currentUser,
  onSelectAuthor,
  gridColumns = 3,
}) {
  // Индикатор загрузки экспозиции (предотвращает мигание экрана "нет фото")
  if (isLoadingPhotos) {
    return (
      <div className={styles.loadingContainer}>
        <div className={styles.spinner}></div>
        <span className={styles.loadingText}>ЗАГРУЗКА ЭКСПОЗИЦИИ...</span>
      </div>
    );
  }

  // Заглушка, если в галерее пока нет ни одного снимка
  if (!photosList || photosList.length === 0) {
    return (
      <div className={styles.emptyContainer}>
        <div className={styles.emptyGraphic}>
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <circle cx="12" cy="10" r="7" strokeWidth="1.5" />
            <circle cx="12" cy="10" r="2.5" fill="currentColor" />
            <path d="M6 20L9 17L12 19L15 17L18 20" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>
        <h2 className={styles.emptyTitle}>ФОТОГРАФИЙ ПОКА НЕТ</h2>
        <p className={styles.emptySubtitle}>
          Опубликуйте первые авторские фотографии в высоком разрешении
        </p>
        {currentUser && (
          <button className={styles.emptyButton} onClick={onOpenUploadModal}>
            ЗАГРУЗИТЬ ПЕРВОЕ ФОТО
          </button>
        )}
      </div>
    );
  }

  // Отрисовка фотографий в настраиваемой адаптивной модульной сетке
  const columnClass = styles[`cols${gridColumns}`] || styles.cols3;

  return (
    <div className={`${styles.grid} ${columnClass}`}>
      {photosList.map((photoItem) => (
        <PhotoCard
          key={photoItem.id}
          photoItem={photoItem}
          onSelectPhoto={onSelectPhoto}
          onDeletePhoto={onDeletePhoto}
          currentUser={currentUser}
          onSelectAuthor={onSelectAuthor}
        />
      ))}
    </div>
  );
}

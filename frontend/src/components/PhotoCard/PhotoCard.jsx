import React from 'react';
import { Trash2 } from 'lucide-react';
import styles from './PhotoCard.module.scss';

/**
 * Компонент карточки фотографии в модульной сетке.
 * Отображает снимок, плавно реагирует на наведение курсора,
 * позволяет удалить снимок (при наличии прав) и открыть полноэкранный просмотр.
 */
export default function PhotoCard({
  photoItem,
  onSelectPhoto,
  onDeletePhoto,
  currentUser,
  onSelectAuthor,
}) {
  // Пользователь имеет право удалить снимок, если он администратор либо автор фото
  const canDeletePhoto =
    currentUser && (currentUser.role === 'admin' || currentUser.id === photoItem.user_id);

  // Обработка удаления фото с подтверждением пользователя
  const handleDeleteButtonClick = (clickEvent) => {
    clickEvent.stopPropagation(); // Предотвращаем открытие Lightbox
    if (window.confirm('Вы действительно хотите удалить эту фотографию?')) {
      onDeletePhoto(photoItem.id);
    }
  };

  return (
    <div className={styles.card} onClick={() => onSelectPhoto(photoItem)}>
      <div className={styles.imageWrapper}>
        <img
          src={photoItem.file_path_medium || photoItem.file_path_original}
          alt={photoItem.title || 'Фотография'}
          loading="lazy"
          className={styles.image}
        />
        <div className={styles.overlay}>
          {canDeletePhoto && (
            <div className={styles.topRow}>
              <button
                className={styles.deleteButton}
                onClick={handleDeleteButtonClick}
                title="Удалить фотографию"
              >
                <Trash2 size={15} strokeWidth={1.8} />
              </button>
            </div>
          )}

          <div className={styles.bottomRow}>
            <h3 className={styles.title}>{photoItem.title}</h3>
            {photoItem.author_username && (
              <span
                className={styles.author}
                onClick={(clickEvent) => {
                  if (onSelectAuthor) {
                    clickEvent.stopPropagation();
                    onSelectAuthor({
                      id: photoItem.user_id,
                      username: photoItem.author_username,
                    });
                  }
                }}
                style={{ cursor: onSelectAuthor ? 'pointer' : 'default' }}
                title={onSelectAuthor ? `Все работы @${photoItem.author_username}` : ''}
              >
                @{photoItem.author_username}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

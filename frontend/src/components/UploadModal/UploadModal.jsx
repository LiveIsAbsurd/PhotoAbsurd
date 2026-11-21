import React, { useState, useRef, useEffect } from 'react';
import { X, UploadCloud, CheckCircle, AlertCircle } from 'lucide-react';
import styles from './UploadModal.module.scss';
import { uploadPhotosPayload } from '../../services/api';

/**
 * Модальное окно для публикации фотографий.
 * Поддерживает Drag & Drop, пакетную загрузку файлов,
 * выбор альбома и валидацию перед отправкой.
 */
export default function UploadModal({
  isModalOpen,
  onCloseModal,
  albumsList,
  defaultAlbumId = '',
  onUploadSuccess,
  onRequireAuthentication,
  currentUser,
}) {
  const [selectedFilesList, setSelectedFilesList] = useState([]);
  const [photoTitleInput, setPhotoTitleInput] = useState('');
  const [photoDescriptionInput, setPhotoDescriptionInput] = useState('');
  const [selectedAlbumId, setSelectedAlbumId] = useState(defaultAlbumId || '');
  const [isUploadInProgress, setIsUploadInProgress] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);
  const [isSuccessMessageVisible, setIsSuccessMessageVisible] = useState(false);

  const hiddenFileInputReference = useRef(null);

  // Синхронизируем выбранный альбом при каждом открытии окна
  useEffect(() => {
    if (isModalOpen) {
      setSelectedAlbumId(defaultAlbumId || '');
      setErrorMessage(null);
    }
  }, [isModalOpen, defaultAlbumId]);

  if (!isModalOpen) return null;

  // Если пользователь не авторизован — показываем требование войти в аккаунт
  if (!currentUser) {
    return (
      <div className={styles.overlay} onClick={onCloseModal}>
        <div className={styles.modal} onClick={(clickEvent) => clickEvent.stopPropagation()}>
          <header className={styles.header}>
            <span className={styles.title}>ТРЕБУЕТСЯ АВТОРИЗАЦИЯ</span>
            <button className={styles.closeBtn} onClick={onCloseModal}>
              <X size={18} />
            </button>
          </header>
          <div className={styles.body}>
            <p className={styles.authPrompt}>
              Публикация авторских фотографий доступна только авторизованным пользователям.
            </p>
            <button
              className={styles.submitBtn}
              onClick={() => {
                onCloseModal();
                onRequireAuthentication();
              }}
            >
              ВОЙТИ В АККАУНТ
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Обработка выбора файлов через стандартный диалог проводника
  const handleFileInputChange = (changeEvent) => {
    if (changeEvent.target.files) {
      setSelectedFilesList(Array.from(changeEvent.target.files));
      setErrorMessage(null);
    }
  };

  // Обработка сброса файлов методом Drag & Drop
  const handleDropFiles = (dragEvent) => {
    dragEvent.preventDefault();
    if (dragEvent.dataTransfer.files) {
      setSelectedFilesList(Array.from(dragEvent.dataTransfer.files));
      setErrorMessage(null);
    }
  };

  // Отправка формы загрузки на сервер
  const handleFormSubmit = async (submitEvent) => {
    submitEvent.preventDefault();
    if (selectedFilesList.length === 0) {
      setErrorMessage('Пожалуйста, выберите хотя бы одну фотографию');
      return;
    }

    setIsUploadInProgress(true);
    setErrorMessage(null);

    const formData = new FormData();
    selectedFilesList.forEach((fileItem) => {
      formData.append('files', fileItem);
    });

    if (photoTitleInput) formData.append('title', photoTitleInput);
    if (photoDescriptionInput) formData.append('description', photoDescriptionInput);
    if (selectedAlbumId) {
      formData.append('album_id', selectedAlbumId);
      // Если выбран приватный альбом, снимок сохраняется как приватный
      const chosenAlbum = albumsList?.find((item) => item.id === selectedAlbumId);
      if (chosenAlbum && chosenAlbum.is_public === false) {
        formData.append('is_public', 'false');
      }
    }

    try {
      await uploadPhotosPayload(formData);
      setIsSuccessMessageVisible(true);

      setTimeout(() => {
        setIsUploadInProgress(false);
        setIsSuccessMessageVisible(false);
        setSelectedFilesList([]);
        setPhotoTitleInput('');
        setPhotoDescriptionInput('');
        onUploadSuccess();
        onCloseModal();
      }, 1000);
    } catch (uploadError) {
      setIsUploadInProgress(false);
      setErrorMessage(uploadError.message || 'Ошибка загрузки файлов');
    }
  };

  return (
    <div className={styles.overlay} onClick={onCloseModal}>
      <div className={styles.modal} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <header className={styles.header}>
          <span className={styles.title}>ЗАГРУЗИТЬ ФОТОГРАФИИ</span>
          <button className={styles.closeBtn} onClick={onCloseModal} aria-label="Закрыть окно">
            <X size={18} />
          </button>
        </header>

        <form onSubmit={handleFormSubmit} className={styles.form}>
          {/* Область Drag & Drop */}
          <div
            className={styles.dropZone}
            onDragOver={(dragEvent) => dragEvent.preventDefault()}
            onDrop={handleDropFiles}
            onClick={() => hiddenFileInputReference.current?.click()}
          >
            <input
              type="file"
              ref={hiddenFileInputReference}
              onChange={handleFileInputChange}
              multiple
              accept="image/jpeg,image/png,image/webp"
              className={styles.hiddenInput}
            />
            <UploadCloud size={36} strokeWidth={1.2} className={styles.uploadIcon} />
            {selectedFilesList.length > 0 ? (
              <div className={styles.fileList}>
                <span className={styles.filesCount}>
                  Выбрано файлов: {selectedFilesList.length}
                </span>
                <span className={styles.fileNames}>
                  {selectedFilesList.map((file) => file.name).join(', ')}
                </span>
              </div>
            ) : (
              <>
                <span className={styles.dropText}>ПЕРЕТАЩИТЕ ФОТОГРАФИИ СЮДА</span>
                <span className={styles.dropSubtext}>JPEG, PNG или WebP · Высокое разрешение</span>
              </>
            )}
          </div>

          {/* Поле названия */}
          <div className={styles.fieldGroup}>
            <label className={styles.label}>НАЗВАНИЕ (НЕОБЯЗАТЕЛЬНО)</label>
            <input
              type="text"
              value={photoTitleInput}
              onChange={(changeEvent) => setPhotoTitleInput(changeEvent.target.value)}
              placeholder="Например: Архитектура тишины"
              className={styles.input}
            />
          </div>

          {/* Выбор альбома */}
          {albumsList && albumsList.length > 0 && (
            <div className={styles.fieldGroup}>
              <label className={styles.label}>ДОБАВИТЬ В АЛЬБОМ (НЕОБЯЗАТЕЛЬНО)</label>
              <select
                value={selectedAlbumId}
                onChange={(changeEvent) => setSelectedAlbumId(changeEvent.target.value)}
                className={styles.select}
              >
                <option value="">Без альбома</option>
                {albumsList.map((albumItem) => (
                  <option key={albumItem.id} value={albumItem.id}>
                    {albumItem.title}
                  </option>
                ))}
              </select>
            </div>
          )}

          {errorMessage && (
            <div className={styles.errorBanner}>
              <AlertCircle size={15} />
              <span>{errorMessage}</span>
            </div>
          )}

          {isSuccessMessageVisible && (
            <div className={styles.successBanner}>
              <CheckCircle size={15} />
              <span>Фотографии успешно опубликованы!</span>
            </div>
          )}

          <button
            type="submit"
            disabled={isUploadInProgress || selectedFilesList.length === 0}
            className={styles.submitBtn}
          >
            {isUploadInProgress ? 'ОБРАБОТКА И ОПТИМИЗАЦИЯ...' : 'ОПУБЛИКОВАТЬ В ГАЛЕРЕЕ'}
          </button>
        </form>
      </div>
    </div>
  );
}

import React, { useState } from 'react';
import { X, FolderPlus, AlertCircle, CheckCircle } from 'lucide-react';
import styles from './CreateAlbumModal.module.scss';
import { createNewAlbum } from '../../services/api';

/**
 * Модальное окно создания нового альбома (коллекции).
 */
export default function CreateAlbumModal({ isModalOpen, onCloseModal, onCreateAlbumSuccess }) {
  const [albumTitleInput, setAlbumTitleInput] = useState('');
  const [albumDescriptionInput, setAlbumDescriptionInput] = useState('');
  const [isAlbumPublic, setIsAlbumPublic] = useState(true);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState(null);
  const [successMessage, setSuccessMessage] = useState(null);

  if (!isModalOpen) return null;

  const handleFormSubmit = async (submitEvent) => {
    submitEvent.preventDefault();
    if (!albumTitleInput.trim()) {
      setErrorMessage('Название альбома обязательно');
      return;
    }

    setIsSubmitting(true);
    setErrorMessage(null);
    setSuccessMessage(null);

    try {
      const createdAlbum = await createNewAlbum({
        albumTitle: albumTitleInput.trim(),
        albumDescription: albumDescriptionInput.trim(),
        isPublic: isAlbumPublic,
      });

      setSuccessMessage(`Альбом «${createdAlbum.title}» успешно создан!`);
      setAlbumTitleInput('');
      setAlbumDescriptionInput('');
      setIsAlbumPublic(true);

      setTimeout(() => {
        setIsSubmitting(false);
        setSuccessMessage(null);
        onCreateAlbumSuccess(createdAlbum);
        onCloseModal();
      }, 700);
    } catch (createError) {
      setIsSubmitting(false);
      setErrorMessage(createError.message || 'Ошибка создания альбома');
    }
  };

  return (
    <div className={styles.overlay} onClick={onCloseModal}>
      <div className={styles.modal} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <header className={styles.header}>
          <div className={styles.titleGroup}>
            <FolderPlus size={18} strokeWidth={2} />
            <span className={styles.title}>СОЗДАТЬ АЛЬБОМ</span>
          </div>
          <button className={styles.closeBtn} onClick={onCloseModal} aria-label="Закрыть">
            <X size={18} />
          </button>
        </header>

        <form onSubmit={handleFormSubmit} className={styles.form}>
          <div className={styles.fieldGroup}>
            <label className={styles.label}>НАЗВАНИЕ АЛЬБОМА</label>
            <input
              type="text"
              required
              autoFocus
              value={albumTitleInput}
              onChange={(changeEvent) => setAlbumTitleInput(changeEvent.target.value)}
              placeholder="Например: Чёрно-белый портрет"
              className={styles.input}
            />
          </div>

          <div className={styles.fieldGroup}>
            <label className={styles.label}>ОПИСАНИЕ (НЕОБЯЗАТЕЛЬНО)</label>
            <textarea
              rows={3}
              value={albumDescriptionInput}
              onChange={(changeEvent) => setAlbumDescriptionInput(changeEvent.target.value)}
              placeholder="Краткое описание тематики снимков..."
              className={styles.textarea}
            />
          </div>

          <div className={styles.checkboxRow}>
            <label className={styles.checkboxLabel}>
              <input
                type="checkbox"
                checked={isAlbumPublic}
                onChange={(changeEvent) => setIsAlbumPublic(changeEvent.target.checked)}
                className={styles.checkbox}
              />
              <span>Публичный альбом (виден всем посетителям галереи)</span>
            </label>
          </div>

          {errorMessage && (
            <div className={styles.errorBanner}>
              <AlertCircle size={15} />
              <span>{errorMessage}</span>
            </div>
          )}

          {successMessage && (
            <div className={styles.successBanner}>
              <CheckCircle size={15} />
              <span>{successMessage}</span>
            </div>
          )}

          <button type="submit" disabled={isSubmitting} className={styles.submitBtn}>
            {isSubmitting ? 'СОЗДАНИЕ...' : 'СОЗДАТЬ КОЛЛЕКЦИЮ'}
          </button>
        </form>
      </div>
    </div>
  );
}

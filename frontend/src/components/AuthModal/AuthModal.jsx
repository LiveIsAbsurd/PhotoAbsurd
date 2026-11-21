import React, { useState } from 'react';
import { X, AlertCircle } from 'lucide-react';
import styles from './AuthModal.module.scss';
import { authenticateUser } from '../../services/api';

/**
 * Модальное окно авторизации.
 * Принимает логин и пароль, отправляет запрос на сервер
 * и возвращает профиль пользователя в родительский компонент.
 */
export default function AuthModal({ isModalOpen, onCloseModal, onAuthenticationSuccess }) {
  const [usernameInput, setUsernameInput] = useState('');
  const [passwordInput, setPasswordInput] = useState('');
  const [errorMessage, setErrorMessage] = useState(null);
  const [isAuthenticationInProgress, setIsAuthenticationInProgress] = useState(false);

  if (!isModalOpen) return null;

  const handleFormSubmit = async (submitEvent) => {
    submitEvent.preventDefault();
    setIsAuthenticationInProgress(true);
    setErrorMessage(null);

    try {
      const authenticatedUser = await authenticateUser(usernameInput.trim(), passwordInput);
      onAuthenticationSuccess(authenticatedUser);
      onCloseModal();
    } catch (authError) {
      setErrorMessage(authError.message || 'Ошибка авторизации');
    } finally {
      setIsAuthenticationInProgress(false);
    }
  };

  return (
    <div className={styles.overlay} onClick={onCloseModal}>
      <div className={styles.modal} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <header className={styles.header}>
          <span className={styles.title}>ВХОД В ГАЛЕРЕЮ</span>
          <button className={styles.closeBtn} onClick={onCloseModal} aria-label="Закрыть">
            <X size={18} />
          </button>
        </header>

        <form onSubmit={handleFormSubmit} className={styles.form}>
          <div className={styles.fieldGroup}>
            <label className={styles.label}>ЛОГИН</label>
            <input
              type="text"
              required
              autoFocus
              value={usernameInput}
              onChange={(changeEvent) => setUsernameInput(changeEvent.target.value)}
              placeholder="Введите ваш логин"
              className={styles.input}
            />
          </div>

          <div className={styles.fieldGroup}>
            <label className={styles.label}>ПАРОЛЬ</label>
            <input
              type="password"
              required
              value={passwordInput}
              onChange={(changeEvent) => setPasswordInput(changeEvent.target.value)}
              placeholder="••••••••"
              className={styles.input}
            />
          </div>

          {errorMessage && (
            <div className={styles.errorBanner}>
              <AlertCircle size={15} />
              <span>{errorMessage}</span>
            </div>
          )}

          <button
            type="submit"
            disabled={isAuthenticationInProgress}
            className={styles.submitBtn}
          >
            {isAuthenticationInProgress ? 'ПРОВЕРКА ДАННЫХ...' : 'ВОЙТИ'}
          </button>
        </form>
      </div>
    </div>
  );
}

import React, { useState, useEffect } from 'react';
import {
  X,
  UserPlus,
  Trash2,
  Shield,
  User,
  FolderLock,
  Lock,
  Globe,
  AlertCircle,
  CheckCircle,
  KeyRound,
} from 'lucide-react';
import styles from './AdminModal.module.scss';
import {
  fetchAdminUsersList,
  createAdminUser,
  deleteAdminUser,
  adminChangeUserPassword,
  fetchAlbumsList,
  deleteAlbumById,
  updateAlbumAccess,
} from '../../services/api';

/**
 * Панель администратора (AdminModal).
 * Позволяет:
 * 1. Управлять учётными записями авторов (просмотр, создание, удаление).
 * 2. Управлять альбомами и доступом: удалять альбомы и ограничивать доступ
 *    (переключение между публичным режимом и ограниченным доступом).
 */
export default function AdminModal({ isModalOpen, onCloseModal, currentUser, onAlbumsChanged }) {
  const [activeTab, setActiveTab] = useState('users'); // 'users' | 'albums'

  // Состояние раздела пользователей
  const [usersList, setUsersList] = useState([]);
  const [newUsernameInput, setNewUsernameInput] = useState('');
  const [newPasswordInput, setNewPasswordInput] = useState('');
  const [newRoleSelection, setNewRoleSelection] = useState('user');
  const [isLoadingUsersList, setIsLoadingUsersList] = useState(false);
  const [isCreatingUserInProgress, setIsCreatingUserInProgress] = useState(false);

  // Состояние смены пароля пользователя
  const [editingPasswordUserId, setEditingPasswordUserId] = useState(null);
  const [newPasswordForUser, setNewPasswordForUser] = useState('');
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false);

  // Состояние раздела альбомов
  const [albumsList, setAlbumsList] = useState([]);
  const [isLoadingAlbums, setIsLoadingAlbums] = useState(false);

  // Баннеры статуса операций
  const [operationErrorMessage, setOperationErrorMessage] = useState(null);
  const [operationSuccessMessage, setOperationSuccessMessage] = useState(null);

  // Загружаем данные при открытии модального окна
  useEffect(() => {
    if (isModalOpen) {
      loadRegisteredUsers();
      loadAlbums();
    }
  }, [isModalOpen]);

  // Загрузка пользователей
  const loadRegisteredUsers = async () => {
    setIsLoadingUsersList(true);
    setOperationErrorMessage(null);
    try {
      const receivedUsersData = await fetchAdminUsersList();
      setUsersList(Array.isArray(receivedUsersData) ? receivedUsersData : []);
    } catch (fetchError) {
      setUsersList([]);
      console.warn('Не удалось загрузить список пользователей:', fetchError);
    } finally {
      setIsLoadingUsersList(false);
    }
  };

  // Загрузка всех альбомов
  const loadAlbums = async () => {
    setIsLoadingAlbums(true);
    try {
      const receivedAlbums = await fetchAlbumsList();
      setAlbumsList(Array.isArray(receivedAlbums) ? receivedAlbums : []);
    } catch (albumsError) {
      console.warn('Ошибка загрузки альбомов:', albumsError);
    } finally {
      setIsLoadingAlbums(false);
    }
  };

  // Создание нового пользователя
  const handleCreateUserSubmit = async (submitEvent) => {
    submitEvent.preventDefault();
    setIsCreatingUserInProgress(true);
    setOperationErrorMessage(null);
    setOperationSuccessMessage(null);

    try {
      await createAdminUser({
        username: newUsernameInput,
        password: newPasswordInput,
        role: newRoleSelection,
      });

      setOperationSuccessMessage(`Пользователь @${newUsernameInput} успешно создан!`);
      setNewUsernameInput('');
      setNewPasswordInput('');
      setNewRoleSelection('user');

      await loadRegisteredUsers();
    } catch (createError) {
      setOperationErrorMessage(createError.message || 'Ошибка создания пользователя');
    } finally {
      setIsCreatingUserInProgress(false);
    }
  };

  // Удаление пользователя
  const handleDeleteUserClick = async (targetUserId, targetUsername) => {
    if (!window.confirm(`Вы уверены, что хотите удалить пользователя @${targetUsername}?`)) {
      return;
    }

    try {
      await deleteAdminUser(targetUserId);
      setUsersList((previousList) => previousList.filter((userItem) => userItem.id !== targetUserId));
      setOperationSuccessMessage(`Пользователь @${targetUsername} удалён`);
    } catch (deleteError) {
      setOperationErrorMessage(deleteError.message || 'Ошибка удаления пользователя');
    }
  };

  // Смена пароля пользователю администратором
  const handleStartChangePassword = (userId) => {
    setEditingPasswordUserId(userId);
    setNewPasswordForUser('');
    setOperationErrorMessage(null);
  };

  const handleCancelChangePassword = () => {
    setEditingPasswordUserId(null);
    setNewPasswordForUser('');
  };

  const handleSavePasswordSubmit = async (userId, username) => {
    if (!newPasswordForUser || newPasswordForUser.length < 6) {
      setOperationErrorMessage('Пароль должен содержать не менее 6 символов');
      return;
    }

    setIsUpdatingPassword(true);
    setOperationErrorMessage(null);
    setOperationSuccessMessage(null);

    try {
      await adminChangeUserPassword(userId, newPasswordForUser);
      setOperationSuccessMessage(`Пароль для @${username} успешно изменён! Все его активные сессии сброшены.`);
      setEditingPasswordUserId(null);
      setNewPasswordForUser('');
    } catch (err) {
      setOperationErrorMessage(err.message || 'Ошибка смены пароля');
    } finally {
      setIsUpdatingPassword(false);
    }
  };

  // Переключение доступа к альбому (публичный / ограниченный)
  const handleToggleAlbumAccessClick = async (targetAlbumId, currentIsPublic) => {
    const nextIsPublic = !currentIsPublic;
    try {
      await updateAlbumAccess(targetAlbumId, nextIsPublic);
      setAlbumsList((previousAlbums) =>
        previousAlbums.map((albumItem) =>
          albumItem.id === targetAlbumId ? { ...albumItem, is_public: nextIsPublic } : albumItem
        )
      );
      setOperationSuccessMessage(
        nextIsPublic ? 'Альбом теперь публичен для всех посетителей' : 'Доступ к альбому ограничен'
      );
      if (onAlbumsChanged) onAlbumsChanged();
    } catch (accessError) {
      setOperationErrorMessage(accessError.message || 'Ошибка изменения доступа к альбому');
    }
  };

  // Удаление альбома администратором
  const handleDeleteAlbumClick = async (targetAlbumId, albumTitle) => {
    if (
      !window.confirm(
        `Вы уверены, что хотите удалить коллекцию «${albumTitle}»? Все фотографии останутся в галерее.`
      )
    ) {
      return;
    }

    try {
      await deleteAlbumById(targetAlbumId);
      setAlbumsList((previousAlbums) =>
        previousAlbums.filter((albumItem) => albumItem.id !== targetAlbumId)
      );
      setOperationSuccessMessage(`Альбом «${albumTitle}» успешно удалён`);
      if (onAlbumsChanged) onAlbumsChanged();
    } catch (deleteError) {
      setOperationErrorMessage(deleteError.message || 'Ошибка удаления альбома');
    }
  };

  if (!isModalOpen) return null;

  return (
    <div className={styles.overlay} onClick={onCloseModal}>
      <div className={styles.modal} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        {/* Шапка модального окна */}
        <header className={styles.header}>
          <div className={styles.titleGroup}>
            <Shield size={18} strokeWidth={2} />
            <span className={styles.title}>ПАНЕЛЬ АДМИНИСТРАТОРА</span>
          </div>
          <button className={styles.closeBtn} onClick={onCloseModal} aria-label="Закрыть">
            <X size={18} />
          </button>
        </header>

        {/* Вкладки разделов управления */}
        <div className={styles.tabsBar}>
          <button
            className={`${styles.tabBtn} ${activeTab === 'users' ? styles.activeTab : ''}`}
            onClick={() => {
              setActiveTab('users');
              setOperationErrorMessage(null);
              setOperationSuccessMessage(null);
            }}
          >
            <User size={14} />
            <span>ПОЛЬЗОВАТЕЛИ ({usersList.length})</span>
          </button>
          <button
            className={`${styles.tabBtn} ${activeTab === 'albums' ? styles.activeTab : ''}`}
            onClick={() => {
              setActiveTab('albums');
              setOperationErrorMessage(null);
              setOperationSuccessMessage(null);
            }}
          >
            <FolderLock size={14} />
            <span>АЛЬБОМЫ И ДОСТУП ({albumsList.length})</span>
          </button>
        </div>

        <div className={styles.content}>
          {/* Сообщения о статусе операций */}
          {operationErrorMessage && (
            <div className={styles.errorBanner}>
              <AlertCircle size={15} />
              <span>{operationErrorMessage}</span>
            </div>
          )}

          {operationSuccessMessage && (
            <div className={styles.successBanner}>
              <CheckCircle size={15} />
              <span>{operationSuccessMessage}</span>
            </div>
          )}

          {/* Вкладка 1: Управление пользователями */}
          {activeTab === 'users' && (
            <>
              {/* Форма создания нового пользователя */}
              <section className={styles.createSection}>
                <h3 className={styles.sectionHeading}>
                  <UserPlus size={16} /> ДОБАВИТЬ НОВОГО ПОЛЬЗОВАТЕЛЯ
                </h3>
                <form onSubmit={handleCreateUserSubmit} className={styles.createForm}>
                  <div className={styles.formRow}>
                    <div className={styles.fieldGroup}>
                      <label className={styles.label}>ЛОГИН</label>
                      <input
                        type="text"
                        required
                        value={newUsernameInput}
                        onChange={(changeEvent) => setNewUsernameInput(changeEvent.target.value)}
                        placeholder="Логин пользователя"
                        className={styles.input}
                      />
                    </div>

                    <div className={styles.fieldGroup}>
                      <label className={styles.label}>ПАРОЛЬ</label>
                      <input
                        type="password"
                        required
                        minLength={6}
                        value={newPasswordInput}
                        onChange={(changeEvent) => setNewPasswordInput(changeEvent.target.value)}
                        placeholder="Пароль (мин. 6 знаков)"
                        className={styles.input}
                      />
                    </div>

                    <div className={styles.fieldGroup}>
                      <label className={styles.label}>РОЛЬ</label>
                      <select
                        value={newRoleSelection}
                        onChange={(changeEvent) => setNewRoleSelection(changeEvent.target.value)}
                        className={styles.select}
                      >
                        <option value="user">Пользователь</option>
                        <option value="admin">Администратор</option>
                      </select>
                    </div>
                  </div>

                  <button
                    type="submit"
                    disabled={isCreatingUserInProgress}
                    className={styles.submitBtn}
                  >
                    {isCreatingUserInProgress ? 'СОЗДАНИЕ...' : 'СОЗДАТЬ ПОЛЬЗОВАТЕЛЯ'}
                  </button>
                </form>
              </section>

              {/* Список зарегистрированных пользователей */}
              <section className={styles.listSection}>
                <h3 className={styles.sectionHeading}>
                  <User size={16} /> СПИСОК ПОЛЬЗОВАТЕЛЕЙ ({usersList.length})
                </h3>

                {usersList.length === 0 ? (
                  <div className={styles.emptyUsers}>
                    <span>Пользователей в системе пока нет. Вы можете зарегистрировать первого автора в форме выше.</span>
                  </div>
                ) : (
                  <div className={styles.usersList}>
                    {usersList.map((userItem) => (
                      <div key={userItem.id} className={styles.userRowContainer}>
                        <div className={styles.userRow}>
                          <div className={styles.userInfo}>
                            <span className={styles.userName}>@{userItem.username}</span>
                            <span
                              className={`${styles.roleBadge} ${userItem.role === 'admin' ? styles.adminBadge : ''}`}
                            >
                              {userItem.role === 'admin' ? 'Администратор' : 'Пользователь'}
                            </span>
                          </div>

                          <div className={styles.userActions}>
                            <button
                              className={`${styles.keyBtn} ${editingPasswordUserId === userItem.id ? styles.activeKeyBtn : ''}`}
                              onClick={() =>
                                editingPasswordUserId === userItem.id
                                  ? handleCancelChangePassword()
                                  : handleStartChangePassword(userItem.id)
                              }
                              title="Сменить пароль пользователю"
                            >
                              <KeyRound size={16} />
                            </button>
                            {currentUser?.id !== userItem.id && (
                              <button
                                className={styles.deleteBtn}
                                onClick={() => handleDeleteUserClick(userItem.id, userItem.username)}
                                title="Удалить пользователя"
                              >
                                <Trash2 size={16} />
                              </button>
                            )}
                          </div>
                        </div>

                        {editingPasswordUserId === userItem.id && (
                          <div className={styles.changePasswordRow}>
                            <input
                              type="password"
                              placeholder="Новый пароль (мин. 6 знаков)"
                              value={newPasswordForUser}
                              onChange={(changeEvent) => setNewPasswordForUser(changeEvent.target.value)}
                              className={styles.passwordInput}
                              autoFocus
                            />
                            <button
                              className={styles.savePasswordBtn}
                              disabled={isUpdatingPassword}
                              onClick={() => handleSavePasswordSubmit(userItem.id, userItem.username)}
                            >
                              {isUpdatingPassword ? '...' : 'Сохранить'}
                            </button>
                            <button
                              className={styles.cancelPasswordBtn}
                              onClick={handleCancelChangePassword}
                            >
                              Отмена
                            </button>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </section>
            </>
          )}

          {/* Вкладка 2: Управление альбомами и ограничение доступа */}
          {activeTab === 'albums' && (
            <section className={styles.listSection}>
              <h3 className={styles.sectionHeading}>
                <FolderLock size={16} /> КОЛЛЕКЦИИ И НАСТРОЙКИ ДОСТУПА ({albumsList.length})
              </h3>

              {albumsList.length === 0 ? (
                <div className={styles.emptyUsers}>
                  <span>В системе пока нет созданных альбомов.</span>
                </div>
              ) : (
                <div className={styles.usersList}>
                  {albumsList.map((albumItem) => (
                    <div key={albumItem.id} className={styles.albumRow}>
                      <div className={styles.albumMainInfo}>
                        <div className={styles.albumTitleRow}>
                          <span className={styles.albumTitle}>{albumItem.title}</span>
                          <span className={styles.albumPhotoCount}>
                            {albumItem.photos_count || 0} фото
                          </span>
                        </div>
                        {albumItem.description && (
                          <span className={styles.albumDesc}>{albumItem.description}</span>
                        )}
                      </div>

                      <div className={styles.albumActions}>
                        {albumItem.is_public ? (
                          <span className={`${styles.accessBadge} ${styles.public}`}>
                            <Globe size={12} /> Публичный
                          </span>
                        ) : (
                          <span className={`${styles.accessBadge} ${styles.restricted}`}>
                            <Lock size={12} /> Ограничен
                          </span>
                        )}

                        <button
                          className={styles.toggleAccessBtn}
                          onClick={() =>
                            handleToggleAlbumAccessClick(albumItem.id, albumItem.is_public)
                          }
                          title={
                            albumItem.is_public
                              ? 'Ограничить доступ (сделать приватным)'
                              : 'Открыть публичный доступ'
                          }
                        >
                          {albumItem.is_public ? 'ОГРАНИЧИТЬ' : 'ОТКРЫТЬ'}
                        </button>

                        <button
                          className={styles.deleteAlbumBtn}
                          onClick={() => handleDeleteAlbumClick(albumItem.id, albumItem.title)}
                          title="Удалить альбом"
                        >
                          <Trash2 size={14} />
                          <span>УДАЛИТЬ</span>
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

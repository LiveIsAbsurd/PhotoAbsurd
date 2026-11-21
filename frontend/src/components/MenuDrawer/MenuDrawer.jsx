import React from 'react';
import { X, Upload, Shield, User, LogOut, FolderPlus } from 'lucide-react';
import styles from './MenuDrawer.module.scss';

/**
 * Боковая выезжающая шторка (Drawer) с навигацией:
 * - Сортировка ("Свежие", "Популярные")
 * - Тематические коллекции (альбомы)
 * - Кнопка создания нового альбома
 * - Кнопка быстрой загрузки фотографий
 * - Панель администратора (для пользователей с ролью admin)
 * - Кнопка входа / выхода из профиля
 */
export default function MenuDrawer({
  isDrawerOpen,
  onCloseDrawer,
  currentUser,
  currentSortType,
  onSortTypeChange,
  onOpenUploadModal,
  onOpenAuthModal,
  onOpenAdminModal,
  onOpenCreateAlbumModal,
  onLogoutUser,
  albumsList,
  selectedAlbum,
  onSelectAlbum,
  authorsList,
  selectedAuthor,
  onSelectAuthor,
}) {
  if (!isDrawerOpen) return null;

  // Фильтрация альбомов для меню:
  // - Незалогиненный пользователь видит строго публичные альбомы (is_public === true).
  // - Авторизованный пользователь видит публичные альбомы + созданные им лично.
  // - Администратор видит все альбомы.
  const displayedAlbums = (albumsList || []).filter((albumItem) => {
    if (!currentUser) {
      return albumItem.is_public === true;
    }
    if (currentUser.role === 'admin') {
      return true;
    }
    return albumItem.is_public === true || albumItem.user_id === currentUser.id;
  });

  return (
    <div className={styles.overlay} onClick={onCloseDrawer}>
      <div className={styles.drawer} onClick={(clickEvent) => clickEvent.stopPropagation()}>
        {/* Заголовок шторки */}
        <header className={styles.header}>
          <span className={styles.menuLabel}>РАЗДЕЛЫ</span>
          <button className={styles.closeBtn} onClick={onCloseDrawer} aria-label="Закрыть меню">
            <span className={styles.closeText}>ЗАКРЫТЬ</span>
            <X size={18} strokeWidth={1.5} />
          </button>
        </header>

        {/* Навигационные ссылки */}
        <nav className={styles.nav}>
          {/* Раздел кураторской ленты */}
          <div className={styles.navSection}>
            <span className={styles.sectionTitle}>КУРАТОРСКАЯ ЛЕНТА</span>
            <button
              className={`${styles.navItem} ${!selectedAlbum && !selectedAuthor ? styles.active : ''}`}
              onClick={() => {
                onSelectAlbum(null);
                if (onSelectAuthor) onSelectAuthor(null);
                onSortTypeChange('newest');
                onCloseDrawer();
              }}
            >
              <span className={styles.navItemTitle}>СВЕЖИЕ ФОТОГРАФИИ</span>
            </button>
          </div>

          {/* Список альбомов и создание */}
          <div className={styles.navSection}>
            <div className={styles.sectionHeaderRow}>
              <span className={styles.sectionTitle}>АЛЬБОМЫ И КОЛЛЕКЦИИ</span>
              {currentUser && (
                <button
                  className={styles.createAlbumLink}
                  onClick={() => {
                    onCloseDrawer();
                    onOpenCreateAlbumModal();
                  }}
                  title="Создать новый альбом"
                >
                  <FolderPlus size={14} />
                  <span>СОЗДАТЬ</span>
                </button>
              )}
            </div>

            {displayedAlbums && displayedAlbums.length > 0 ? (
              displayedAlbums.map((albumItem) => (
                <button
                  key={albumItem.id}
                  className={`${styles.navItem} ${selectedAlbum?.id === albumItem.id ? styles.active : ''}`}
                  onClick={() => {
                    if (onSelectAuthor) onSelectAuthor(null);
                    onSelectAlbum(albumItem);
                    onCloseDrawer();
                  }}
                >
                  <span className={styles.navItemTitle}>{albumItem.title}</span>
                  <span className={styles.badge}>{albumItem.photos_count || 0}</span>
                </button>
              ))
            ) : (
              <span className={styles.emptyNotice}>Альбомов пока нет</span>
            )}
          </div>

          {/* Список авторов и публикаторов */}
          {authorsList && authorsList.length > 0 && (
            <div className={styles.navSection}>
              <span className={styles.sectionTitle}>АВТОРЫ И ПУБЛИКАТОРЫ</span>
              {authorsList.map((authorItem) => (
                <button
                  key={authorItem.id}
                  className={`${styles.navItem} ${selectedAuthor?.id === authorItem.id ? styles.active : ''}`}
                  onClick={() => {
                    onSelectAlbum(null);
                    if (onSelectAuthor) onSelectAuthor(authorItem);
                    onCloseDrawer();
                  }}
                >
                  <span className={styles.navItemTitle}>@{authorItem.username}</span>
                  <span className={styles.badge}>{authorItem.photos_count}</span>
                </button>
              ))}
            </div>
          )}

          {/* Действия: публикация фото и админка (только для авторизованных пользователей) */}
          {currentUser && (
            <div className={styles.navSection}>
              <span className={styles.sectionTitle}>ДЕЙСТВИЯ</span>
              <button
                className={styles.actionItem}
                onClick={() => {
                  onCloseDrawer();
                  onOpenUploadModal();
                }}
              >
                <Upload size={16} strokeWidth={1.8} />
                <span>ЗАГРУЗИТЬ ФОТОГРАФИИ</span>
              </button>

              {currentUser?.role === 'admin' && (
                <button
                  className={styles.adminItem}
                  onClick={() => {
                    onCloseDrawer();
                    onOpenAdminModal();
                  }}
                >
                  <Shield size={16} strokeWidth={1.8} />
                  <span>ПАНЕЛЬ АДМИНИСТРАТОРА</span>
                </button>
              )}
            </div>
          )}
        </nav>

        {/* Подвал шторки: статус пользователя или кнопка входа */}
        <footer className={styles.footer}>
          {currentUser ? (
            <div className={styles.userProfile}>
              <div className={styles.userInfo}>
                <span className={styles.userLabel}>
                  {currentUser.role === 'admin' ? 'АДМИНИСТРАТОР' : 'ВЫ ВОШЛИ КАК'}
                </span>
                <span className={styles.userName}>@{currentUser.username}</span>
              </div>
              <button className={styles.logoutBtn} onClick={onLogoutUser} title="Выйти из аккаунта">
                <LogOut size={16} strokeWidth={1.8} />
              </button>
            </div>
          ) : (
            <button
              className={styles.authBtn}
              onClick={() => {
                onCloseDrawer();
                onOpenAuthModal();
              }}
            >
              <User size={16} strokeWidth={1.8} />
              <span>ВХОД</span>
            </button>
          )}
        </footer>
      </div>
    </div>
  );
}

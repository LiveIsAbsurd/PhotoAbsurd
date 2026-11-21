import React, { useState, useEffect, useCallback } from 'react';
import Header from './components/Header/Header';
import PhotoGrid from './components/PhotoGrid/PhotoGrid';
import Lightbox from './components/Lightbox/Lightbox';
import MenuDrawer from './components/MenuDrawer/MenuDrawer';
import UploadModal from './components/UploadModal/UploadModal';
import AuthModal from './components/AuthModal/AuthModal';
import AdminModal from './components/AdminModal/AdminModal';
import CreateAlbumModal from './components/CreateAlbumModal/CreateAlbumModal';
import {
  fetchPhotos,
  fetchAlbumsList,
  fetchAlbumById,
  deletePhotoById,
  fetchCurrentUserProfile,
  logoutUser,
} from './services/api';
import './styles/globals.scss';

/**
 * Извлекает информацию о маршруте (альбоме или авторе) из текущей адресной строки браузера.
 * Поддерживает форматы:
 * - /album/:id или /albums/:id или ?album=:id
 * - /author/:username или /@:username или ?author=:username
 */
function parseRouteFromURL() {
  const pathname = window.location.pathname;
  const searchParams = new URLSearchParams(window.location.search);

  // 1. Query-параметры (?album=ID или ?author=username)
  const albumQuery = searchParams.get('album') || searchParams.get('album_id');
  const authorQuery =
    searchParams.get('author') || searchParams.get('username') || searchParams.get('user_id');

  if (albumQuery) {
    return { type: 'album', id: albumQuery };
  }
  if (authorQuery) {
    return { type: 'author', identifier: authorQuery.replace(/^@/, '') };
  }

  // 2. Пути вида /album/:id или /albums/:id
  const albumMatch = pathname.match(/^\/albums?\/([^/]+)/i);
  if (albumMatch) {
    return { type: 'album', id: decodeURIComponent(albumMatch[1]) };
  }

  // 3. Пути вида /author/:identifier или /@:identifier
  const authorMatch = pathname.match(/^\/author\/([^/]+)/i) || pathname.match(/^\/@([^/]+)/i);
  if (authorMatch) {
    return { type: 'author', identifier: decodeURIComponent(authorMatch[1]) };
  }

  return { type: 'home' };
}

/**
 * Корневой компонент веб-приложения "Фотогалерея".
 * Управляет глобальным состоянием:
 * - Загрузка и фильтрация фотографий
 * - Управление сессией пользователя и администратора
 * - Открытие и закрытие модальных окон
 * - Переключение цветовой темы (светлая/тёмная)
 * - URL-роутинг по авторам и коллекциям
 */
export default function App() {
  // Список отображаемых фотографий
  const [photosCollection, setPhotosCollection] = useState([]);
  // Список доступных альбомов/коллекций
  const [albumsCollection, setAlbumsCollection] = useState([]);
  // Профиль текущего авторизованного пользователя (null, если не авторизован)
  const [currentUserProfile, setCurrentUserProfile] = useState(null);

  // Выбранное фото для полноэкранного просмотра в Lightbox
  const [currentSelectedPhoto, setCurrentSelectedPhoto] = useState(null);
  // Активный выбранный альбом для фильтрации (null — показывать все фото)
  const [currentSelectedAlbum, setCurrentSelectedAlbum] = useState(null);
  // Выбранный автор/публикатор для фильтрации (null — показывать всех)
  const [currentSelectedAuthor, setCurrentSelectedAuthor] = useState(null);
  // Список авторов/публикаторов с количеством их работ
  const [authorsList, setAuthorsList] = useState([]);
  // Текущий тип сортировки: "newest" (свежие) или "popular" (популярные)
  const [currentSortType, setCurrentSortType] = useState('newest');

  // Тема оформления: 'light' или 'dark' (с сохранением в localStorage)
  const [currentTheme, setCurrentTheme] = useState(() => {
    return localStorage.getItem('gallery_theme') || 'light';
  });

  // Синхронизация темы с data-theme на корневом элементе HTML
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', currentTheme);
    localStorage.setItem('gallery_theme', currentTheme);
  }, [currentTheme]);

  const handleToggleTheme = () => {
    setCurrentTheme((previousTheme) => (previousTheme === 'light' ? 'dark' : 'light'));
  };

  // Масштаб сетки (количество колонок в одной строке: от 1 до 6, по умолчанию 3)
  const [gridColumns, setGridColumns] = useState(() => {
    const savedColumns = Number(localStorage.getItem('gallery_grid_columns'));
    return savedColumns >= 1 && savedColumns <= 6 ? savedColumns : 3;
  });

  useEffect(() => {
    localStorage.setItem('gallery_grid_columns', String(gridColumns));
  }, [gridColumns]);

  const handleZoomIn = () => {
    setGridColumns((prev) => Math.max(1, prev - 1));
  };

  const handleZoomOut = () => {
    setGridColumns((prev) => Math.min(6, prev + 1));
  };

  const handleCycleZoom = () => {
    setGridColumns((prev) => (prev >= 6 ? 1 : prev + 1));
  };

  // Флаги видимости модальных окон:
  const [isMenuDrawerOpen, setIsMenuDrawerOpen] = useState(false);
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [isAdminModalOpen, setIsAdminModalOpen] = useState(false);
  const [isCreateAlbumModalOpen, setIsCreateAlbumModalOpen] = useState(false);

  // Состояние загрузки снимков (лоадер вместо мигающей пустой заглушки)
  const [isLoadingPhotos, setIsLoadingPhotos] = useState(true);

  // Синхронизация URL при изменении выбранного альбома или автора
  const updateURL = useCallback((album, author) => {
    if (album && album.id) {
      const targetPath = `/album/${album.id}`;
      if (window.location.pathname !== targetPath || window.location.search !== '') {
        window.history.pushState({ type: 'album', id: album.id }, '', targetPath);
      }
    } else if (author && (author.username || author.id)) {
      const authorIdent = author.username || author.id;
      const targetPath = `/author/${authorIdent}`;
      if (window.location.pathname !== targetPath || window.location.search !== '') {
        window.history.pushState({ type: 'author', identifier: authorIdent }, '', targetPath);
      }
    } else {
      if (window.location.pathname !== '/' || window.location.search !== '') {
        window.history.pushState({ type: 'home' }, '', '/');
      }
    }
  }, []);

  // 1. Первичная загрузка профиля и списка альбомов при старте приложения
  useEffect(() => {
    fetchCurrentUserProfile()
      .then((profileData) => setCurrentUserProfile(profileData))
      .catch((error) => console.warn('Гостевая сессия:', error));

    fetchAlbumsList()
      .then((albumsData) => setAlbumsCollection(albumsData))
      .catch((error) => console.error('Ошибка загрузки альбомов:', error));
  }, []);

  // 2. Роутинг: инициализация фильтров из адресной строки и обработка кнопок Назад/Вперёд
  useEffect(() => {
    const applyRouteFromLocation = async () => {
      const route = parseRouteFromURL();
      if (route.type === 'album') {
        try {
          const albumResponse = await fetchAlbumById(route.id);
          if (albumResponse && albumResponse.album) {
            setCurrentSelectedAlbum(albumResponse.album);
            setCurrentSelectedAuthor(null);
          }
        } catch (albumErr) {
          console.warn('Не удалось загрузить альбом по ссылке:', albumErr);
          setCurrentSelectedAlbum(null);
          window.history.replaceState({ type: 'home' }, '', '/');
        }
      } else if (route.type === 'author') {
        setCurrentSelectedAuthor({ username: route.identifier, id: route.identifier });
        setCurrentSelectedAlbum(null);
      } else {
        setCurrentSelectedAlbum(null);
        setCurrentSelectedAuthor(null);
      }
    };

    applyRouteFromLocation();

    window.addEventListener('popstate', applyRouteFromLocation);
    return () => window.removeEventListener('popstate', applyRouteFromLocation);
  }, []);

  // Переключение выбранного альбома с обновлением адреса
  const handleSelectAlbum = useCallback(
    (album) => {
      setCurrentSelectedAlbum(album);
      if (album) {
        setCurrentSelectedAuthor(null);
        updateURL(album, null);
      } else {
        updateURL(null, null);
      }
    },
    [updateURL]
  );

  // Переключение выбранного автора с обновлением адреса
  const handleSelectAuthor = useCallback(
    (author) => {
      setCurrentSelectedAuthor(author);
      if (author) {
        setCurrentSelectedAlbum(null);
        updateURL(null, author);
      } else {
        updateURL(null, null);
      }
    },
    [updateURL]
  );

  // 3. Загрузка фотографий при смене сортировки, альбома или автора
  const loadPhotosFromBackend = useCallback(async () => {
    setIsLoadingPhotos(true);
    try {
      const isUUID = (str) =>
        typeof str === 'string' &&
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str);

      const filterUserId =
        currentSelectedAuthor && isUUID(currentSelectedAuthor.id)
          ? currentSelectedAuthor.id
          : null;
      const filterUsername = currentSelectedAuthor
        ? currentSelectedAuthor.username && !isUUID(currentSelectedAuthor.username)
          ? currentSelectedAuthor.username
          : !isUUID(currentSelectedAuthor.id)
            ? currentSelectedAuthor.id
            : null
        : null;

      const receivedPhotos = await fetchPhotos({
        sortType: currentSortType,
        filterAlbumId: currentSelectedAlbum ? currentSelectedAlbum.id : null,
        filterUserId,
        filterUsername,
      });

      // Извлекаем список авторов из полной ленты (публичные для гостей, все для администратора)
      if (!currentSelectedAuthor && !currentSelectedAlbum && receivedPhotos) {
        const authorsMap = new Map();
        receivedPhotos.forEach((photo) => {
          if (photo.user_id && photo.author_username) {
            if (currentUserProfile?.role === 'admin' || photo.is_public !== false) {
              if (!authorsMap.has(photo.user_id)) {
                authorsMap.set(photo.user_id, {
                  id: photo.user_id,
                  username: photo.author_username,
                  photos_count: 1,
                });
              } else {
                authorsMap.get(photo.user_id).photos_count += 1;
              }
            }
          }
        });
        setAuthorsList(Array.from(authorsMap.values()));
      }

      // Клиентская фильтрация: администратор и автор снимков видят в том числе непубличные фото
      const canSeePrivate =
        currentUserProfile?.role === 'admin' ||
        (currentUserProfile?.id && currentUserProfile.id === currentSelectedAuthor?.id);

      let displayPhotos = receivedPhotos || [];
      if (currentSelectedAuthor) {
        displayPhotos = displayPhotos.filter((photo) => {
          const isMatchAuthor =
            (currentSelectedAuthor.id && photo.user_id === currentSelectedAuthor.id) ||
            (currentSelectedAuthor.username && photo.author_username === currentSelectedAuthor.username) ||
            (currentSelectedAuthor.id && photo.author_username === currentSelectedAuthor.id);
          return isMatchAuthor && (canSeePrivate || photo.is_public !== false);
        });

        // Если автор был передан по ссылке через UUID или неполный объект, дополняем логин
        if (displayPhotos.length > 0) {
          const matchingPhoto = displayPhotos.find((p) => p.author_username);
          if (
            matchingPhoto &&
            (!currentSelectedAuthor.username ||
              currentSelectedAuthor.username === currentSelectedAuthor.id)
          ) {
            setCurrentSelectedAuthor((prev) => ({
              ...prev,
              id: matchingPhoto.user_id,
              username: matchingPhoto.author_username,
            }));
          }
        }
      }
      setPhotosCollection(displayPhotos);
    } catch (fetchError) {
      console.error('Ошибка получения фотографий:', fetchError);
    } finally {
      setIsLoadingPhotos(false);
    }
  }, [currentSortType, currentSelectedAlbum, currentSelectedAuthor, currentUserProfile]);

  useEffect(() => {
    loadPhotosFromBackend();
  }, [loadPhotosFromBackend]);

  // 3. Обработка удаления фотографии
  const handleDeletePhoto = async (photoId) => {
    try {
      await deletePhotoById(photoId);
      // Удаляем снимок из локального состояния сетки
      setPhotosCollection((previousPhotos) =>
        previousPhotos.filter((photoItem) => photoItem.id !== photoId)
      );
      // Если удалённое фото сейчас открыто в Lightbox — закрываем его
      if (currentSelectedPhoto?.id === photoId) {
        setCurrentSelectedPhoto(null);
      }
    } catch (deleteError) {
      alert(deleteError.message || 'Ошибка удаления фотографии');
    }
  };

  // 5. Выход пользователя из системы
  const handleUserLogout = async () => {
    await logoutUser();
    setCurrentUserProfile(null);
    setCurrentSelectedAlbum((previousAlbum) =>
      previousAlbum && !previousAlbum.is_public ? null : previousAlbum
    );
    fetchAlbumsList()
      .then((publicAlbums) => setAlbumsCollection(publicAlbums || []))
      .catch(() => setAlbumsCollection([]));
    setIsMenuDrawerOpen(false);
  };

  // 6. Навигация в режиме Lightbox (Предыдущее / Следующее фото)
  const currentPhotoIndex = currentSelectedPhoto
    ? photosCollection.findIndex((photoItem) => photoItem.id === currentSelectedPhoto.id)
    : -1;
  const hasPreviousPhoto = currentPhotoIndex > 0;
  const hasNextPhoto = currentPhotoIndex >= 0 && currentPhotoIndex < photosCollection.length - 1;

  const handleNavigateToPreviousPhoto = () => {
    if (hasPreviousPhoto) {
      setCurrentSelectedPhoto(photosCollection[currentPhotoIndex - 1]);
    }
  };

  const handleNavigateToNextPhoto = () => {
    if (hasNextPhoto) {
      setCurrentSelectedPhoto(photosCollection[currentPhotoIndex + 1]);
    }
  };

  return (
    <div className="app-container">
      {/* Главный верхний хедер */}
      <Header
        onOpenMenuDrawer={() => setIsMenuDrawerOpen(true)}
        onLogoClick={() => {
          handleSelectAlbum(null);
          handleSelectAuthor(null);
          setCurrentSortType('newest');
        }}
        currentTheme={currentTheme}
        onToggleTheme={handleToggleTheme}
        gridColumns={gridColumns}
        onZoomIn={handleZoomIn}
        onZoomOut={handleZoomOut}
        onCycleZoom={handleCycleZoom}
      />

      {/* Информационная плашка активного альбома (если выбран конкретный альбом) */}
      {currentSelectedAlbum && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 24px',
            backgroundColor: 'var(--color-bg-alt, #fafafa)',
            borderBottom: '1px solid var(--color-border-light, #f0f0f0)',
            color: 'var(--color-text-primary, #111111)',
          }}
        >
          <div>
            <span
              style={{
                fontSize: '10px',
                fontWeight: 700,
                letterSpacing: '2px',
                color: 'var(--color-text-muted, #888)',
              }}
            >
              КОЛЛЕКЦИЯ /
            </span>
            <span
              style={{
                marginLeft: '6px',
                fontSize: '12px',
                fontWeight: 700,
                letterSpacing: '1px',
                textTransform: 'uppercase',
              }}
            >
              {currentSelectedAlbum.title}
            </span>
          </div>
          <button
            onClick={() => handleSelectAlbum(null)}
            style={{
              fontSize: '10px',
              fontWeight: 700,
              letterSpacing: '1.5px',
              textTransform: 'uppercase',
              color: 'var(--color-text-secondary, #666)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
            }}
          >
            ✕ ВСЕ ФОТОГРАФИИ
          </button>
        </div>
      )}

      {/* Информационная плашка активного публикатора (если выбран конкретный автор) */}
      {currentSelectedAuthor && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 24px',
            backgroundColor: 'var(--color-bg-alt, #fafafa)',
            borderBottom: '1px solid var(--color-border-light, #f0f0f0)',
            color: 'var(--color-text-primary, #111111)',
          }}
        >
          <div>
            <span
              style={{
                fontSize: '10px',
                fontWeight: 700,
                letterSpacing: '2px',
                color: 'var(--color-text-muted, #888)',
              }}
            >
              ПУБЛИКАТОР /
            </span>
            <span
              style={{
                marginLeft: '6px',
                fontSize: '12px',
                fontWeight: 700,
                letterSpacing: '1px',
                textTransform: 'uppercase',
              }}
            >
              @{currentSelectedAuthor.username}
            </span>
          </div>
          <button
            onClick={() => handleSelectAuthor(null)}
            style={{
              fontSize: '10px',
              fontWeight: 700,
              letterSpacing: '1.5px',
              textTransform: 'uppercase',
              color: 'var(--color-text-secondary, #666)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
            }}
          >
            ✕ ВСЕ ФОТОГРАФИИ
          </button>
        </div>
      )}

      {/* Основная модульная сетка фотографий */}
      <main>
        <PhotoGrid
          photosList={photosCollection}
          isLoadingPhotos={isLoadingPhotos}
          onSelectPhoto={(photoItem) => setCurrentSelectedPhoto(photoItem)}
          onOpenUploadModal={() => setIsUploadModalOpen(true)}
          onDeletePhoto={handleDeletePhoto}
          currentUser={currentUserProfile}
          onSelectAuthor={handleSelectAuthor}
          gridColumns={gridColumns}
        />
      </main>

      {/* Полноэкранный просмотр Lightbox */}
      {currentSelectedPhoto && (
        <Lightbox
          currentPhoto={currentSelectedPhoto}
          onCloseLightbox={() => setCurrentSelectedPhoto(null)}
          onNavigateToPreviousPhoto={handleNavigateToPreviousPhoto}
          onNavigateToNextPhoto={handleNavigateToNextPhoto}
          hasPreviousPhoto={hasPreviousPhoto}
          hasNextPhoto={hasNextPhoto}
          onDeletePhoto={handleDeletePhoto}
          currentUser={currentUserProfile}
          onSelectAuthor={handleSelectAuthor}
        />
      )}

      {/* Боковая шторка с разделами и навигацией */}
      <MenuDrawer
        isDrawerOpen={isMenuDrawerOpen}
        onCloseDrawer={() => setIsMenuDrawerOpen(false)}
        currentUser={currentUserProfile}
        currentSortType={currentSortType}
        onSortTypeChange={setCurrentSortType}
        onOpenUploadModal={() => setIsUploadModalOpen(true)}
        onOpenAuthModal={() => setIsAuthModalOpen(true)}
        onOpenAdminModal={() => setIsAdminModalOpen(true)}
        onOpenCreateAlbumModal={() => setIsCreateAlbumModalOpen(true)}
        onLogoutUser={handleUserLogout}
        albumsList={albumsCollection}
        selectedAlbum={currentSelectedAlbum}
        onSelectAlbum={handleSelectAlbum}
        authorsList={authorsList}
        selectedAuthor={currentSelectedAuthor}
        onSelectAuthor={handleSelectAuthor}
      />

      {/* Модальное окно создания нового альбома */}
      <CreateAlbumModal
        isModalOpen={isCreateAlbumModalOpen}
        onCloseModal={() => setIsCreateAlbumModalOpen(false)}
        onCreateAlbumSuccess={(newlyCreatedAlbum) => {
          fetchAlbumsList().then((receivedAlbums) => setAlbumsCollection(receivedAlbums));
          handleSelectAlbum(newlyCreatedAlbum);
        }}
      />

      {/* Модальное окно загрузки и публикации фотографий */}
      <UploadModal
        isModalOpen={isUploadModalOpen}
        onCloseModal={() => setIsUploadModalOpen(false)}
        albumsList={albumsCollection}
        defaultAlbumId={currentSelectedAlbum?.id || ''}
        onUploadSuccess={async () => {
          await loadPhotosFromBackend();
          const refreshedAlbums = await fetchAlbumsList();
          setAlbumsCollection(refreshedAlbums || []);
          // Если открыта конкретная коллекция, обновляем её объект со свежим счётчиком
          if (currentSelectedAlbum) {
            const updatedActiveAlbum = refreshedAlbums?.find(
              (albumItem) => albumItem.id === currentSelectedAlbum.id
            );
            if (updatedActiveAlbum) {
              setCurrentSelectedAlbum(updatedActiveAlbum);
            }
          }
        }}
        onRequireAuthentication={() => setIsAuthModalOpen(true)}
        currentUser={currentUserProfile}
      />

      {/* Модальное окно авторизации (Логин + Пароль) */}
      <AuthModal
        isModalOpen={isAuthModalOpen}
        onCloseModal={() => setIsAuthModalOpen(false)}
        onAuthenticationSuccess={(authenticatedUserData) => {
          setCurrentUserProfile(authenticatedUserData);
          fetchAlbumsList().then((receivedAlbums) => setAlbumsCollection(receivedAlbums || []));
          loadPhotosFromBackend();
        }}
      />

      {/* Панель администратора */}
      <AdminModal
        isModalOpen={isAdminModalOpen}
        onCloseModal={() => setIsAdminModalOpen(false)}
        currentUser={currentUserProfile}
        onAlbumsChanged={() => {
          fetchAlbumsList().then((receivedAlbums) => setAlbumsCollection(receivedAlbums || []));
          loadPhotosFromBackend();
        }}
      />
    </div>
  );
}

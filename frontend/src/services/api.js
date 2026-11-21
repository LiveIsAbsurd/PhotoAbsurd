// Базовый путь для всех запросов к бэкенду
const API_BASE_URL = '/api';

/**
 * Получение списка фотографий с параметрами пагинации, сортировки и альбома
 */
export async function fetchPhotos({
  pageNumber = 1,
  limitPerPage = 24,
  sortType = 'newest',
  filterAlbumId = null,
  filterUserId = null,
  filterUsername = null,
} = {}) {
  const requestParameters = new URLSearchParams({
    page: pageNumber,
    limit: limitPerPage,
    sort: sortType,
  });

  if (filterAlbumId) {
    requestParameters.append('album_id', filterAlbumId);
  }

  if (filterUserId) {
    requestParameters.append('user_id', filterUserId);
  }

  if (filterUsername) {
    requestParameters.append('username', filterUsername);
  }

  const apiResponse = await fetch(`${API_BASE_URL}/photos?${requestParameters.toString()}`, {
    cache: 'no-store',
  });
  if (!apiResponse.ok) {
    throw new Error('Не удалось загрузить список фотографий');
  }
  return apiResponse.json();
}

/**
 * Получение детальной информации о конкретной фотографии по её ID
 */
export async function fetchPhotoById(photoId) {
  const apiResponse = await fetch(`${API_BASE_URL}/photos/${photoId}`);
  if (!apiResponse.ok) {
    throw new Error('Не удалось загрузить фотографию');
  }
  return apiResponse.json();
}

/**
 * Удаление фотографии (для автора снимка или администратора)
 */
export async function deletePhotoById(photoId) {
  const apiResponse = await fetch(`${API_BASE_URL}/photos/${photoId}`, {
    method: 'DELETE',
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка удаления фотографии');
  }
  return apiResponse.json();
}

/**
 * Получение списка всех альбомов (коллекций)
 */
export async function fetchAlbumsList() {
  const apiResponse = await fetch(`${API_BASE_URL}/albums`, {
    cache: 'no-store',
  });
  if (!apiResponse.ok) {
    throw new Error('Не удалось загрузить альбомы');
  }
  return apiResponse.json();
}

/**
 * Получение альбома по ID со списком фотографий внутри
 */
export async function fetchAlbumById(albumId) {
  const apiResponse = await fetch(`${API_BASE_URL}/albums/${albumId}`);
  if (!apiResponse.ok) {
    throw new Error('Не удалось загрузить альбом');
  }
  return apiResponse.json();
}

/**
 * Создание нового альбома
 */
export async function createNewAlbum({ albumTitle, albumDescription = '', isPublic = true }) {
  const apiResponse = await fetch(`${API_BASE_URL}/albums`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      title: albumTitle,
      description: albumDescription,
      is_public: isPublic,
    }),
  });
  if (!apiResponse.ok) {
    throw new Error('Не удалось создать альбом');
  }
  return apiResponse.json();
}

/**
 * Удаление альбома (для автора или администратора)
 */
export async function deleteAlbumById(albumId) {
  const apiResponse = await fetch(`${API_BASE_URL}/albums/${albumId}`, {
    method: 'DELETE',
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка удаления альбома');
  }
  return apiResponse.json();
}

/**
 * Обновление параметров видимости и доступа к альбому
 */
export async function updateAlbumAccess(albumId, isPublic) {
  const apiResponse = await fetch(`${API_BASE_URL}/albums/${albumId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_public: isPublic }),
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка обновления доступа');
  }
  return apiResponse.json();
}

/**
 * Пакетная загрузка фотографий на сервер через multipart/form-data
 */
export async function uploadPhotosPayload(formDataPayload) {
  const apiResponse = await fetch(`${API_BASE_URL}/photos`, {
    method: 'POST',
    body: formDataPayload,
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка загрузки фотографии');
  }
  return apiResponse.json();
}

/**
 * Получение данных текущего авторизованного пользователя
 */
export async function fetchCurrentUserProfile() {
  const apiResponse = await fetch(`${API_BASE_URL}/auth/me`, {
    cache: 'no-store',
  });
  if (!apiResponse.ok) {
    return null;
  }
  return apiResponse.json();
}

/**
 * Авторизация по связке логин + пароль
 */
export async function authenticateUser(username, password) {
  const apiResponse = await fetch(`${API_BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Неверный логин или пароль');
  }
  return apiResponse.json();
}

/**
 * Выход из аккаунта (очистка сессионной куки)
 */
export async function logoutUser() {
  const apiResponse = await fetch(`${API_BASE_URL}/auth/logout`, {
    method: 'POST',
  });
  return apiResponse.json();
}

// ==========================================
// МЕТОДЫ ПАНЕЛИ АДМИНИСТРАТОРА
// ==========================================

/**
 * Получение списка всех пользователей
 */
export async function fetchAdminUsersList() {
  const apiResponse = await fetch(`${API_BASE_URL}/admin/users`, {
    cache: 'no-store',
  });
  if (!apiResponse.ok) {
    throw new Error('Ошибка получения списка пользователей');
  }
  return apiResponse.json();
}

/**
 * Создание нового пользователя администратором
 */
export async function createAdminUser({ username, password, role = 'user' }) {
  const apiResponse = await fetch(`${API_BASE_URL}/admin/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password, role }),
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка создания пользователя');
  }
  return apiResponse.json();
}

/**
 * Удаление пользователя администратором
 */
export async function deleteAdminUser(userId) {
  const apiResponse = await fetch(`${API_BASE_URL}/admin/users/${userId}`, {
    method: 'DELETE',
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка удаления пользователя');
  }
  return apiResponse.json();
}

/**
 * Смена пароля пользователя администратором
 */
export async function adminChangeUserPassword(userId, newPassword) {
  const apiResponse = await fetch(`${API_BASE_URL}/admin/users/${userId}/password`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password: newPassword }),
  });
  if (!apiResponse.ok) {
    const errorPayload = await apiResponse.json().catch(() => ({}));
    throw new Error(errorPayload.error || 'Ошибка смены пароля');
  }
  return apiResponse.json();
}

import React from 'react';
import { Sun, Moon, ZoomIn, ZoomOut } from 'lucide-react';
import styles from './Header.module.scss';

/**
 * Минималистичный фиксированный хедер галереи.
 * Включает:
 * - Сдержанный текстовый логотип "ГАЛЕРЕЯ"
 * - Авторский центрированный символ-эмблему
 * - Блок управления плотностью/масштабом сетки (приблизить / отдалить)
 * - Переключатель Light/Dark темы
 * - Кнопку вызова бокового меню "МЕНЮ"
 */
export default function Header({
  onOpenMenuDrawer,
  onLogoClick,
  currentTheme,
  onToggleTheme,
  gridColumns = 3,
  onZoomIn,
  onZoomOut,
  onCycleZoom,
}) {
  return (
    <header className={styles.header}>
      {/* Левая часть: текстовый логотип */}
      <div className={styles.left}>
        <button className={styles.brand} onClick={onLogoClick}>
          ГАЛЕРЕЯ
        </button>
      </div>

      {/* Центральная часть: минималистичная векторная эмблема */}
      <div className={styles.center} onClick={onLogoClick} title="На главную">
        <svg
          className={styles.emblem}
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <circle cx="12" cy="9" r="6" stroke="currentColor" strokeWidth="1.8" />
          <circle cx="12" cy="9" r="2.2" fill="currentColor" />
          <path
            d="M7 19L9.5 16.5L12 18.5L14.5 16.5L17 19"
            stroke="currentColor"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </div>

      {/* Правая часть: масштаб сетки, переключатель темы и кнопка открытия меню */}
      <div className={styles.right}>
        {/* Кнопки управления масштабом сетки (приблизить/отдалить) */}
        <div className={styles.zoomControlGroup}>
          <button
            className={styles.zoomBtn}
            onClick={onZoomIn}
            disabled={gridColumns <= 1}
            aria-label="Приблизить сетку"
            title="Приблизить (крупнее фото, меньше в ряд)"
          >
            <ZoomIn size={15} strokeWidth={1.8} />
          </button>
          <button
            className={styles.zoomIndicator}
            onClick={onCycleZoom}
            aria-label={`Колонок в строке: ${gridColumns}. Нажмите для переключения.`}
            title={`В ряд: ${gridColumns} (клик для циклического переключения)`}
          >
            {gridColumns}
          </button>
          <button
            className={styles.zoomBtn}
            onClick={onZoomOut}
            disabled={gridColumns >= 6}
            aria-label="Отдалить сетку"
            title="Отдалить (мельче фото, больше в ряд)"
          >
            <ZoomOut size={15} strokeWidth={1.8} />
          </button>
        </div>

        <button
          className={styles.themeToggleBtn}
          onClick={onToggleTheme}
          aria-label={currentTheme === 'dark' ? 'Включить светлую тему' : 'Включить тёмную тему'}
          title={currentTheme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}
        >
          {currentTheme === 'dark' ? (
            <Sun size={15} strokeWidth={1.8} />
          ) : (
            <Moon size={15} strokeWidth={1.8} />
          )}
        </button>

        <button
          className={styles.menuButton}
          onClick={onOpenMenuDrawer}
          aria-label="Открыть меню"
        >
          <span className={styles.menuText}>МЕНЮ</span>
          <span className={styles.menuIcon}>
            <span className={styles.line}></span>
            <span className={styles.line}></span>
          </span>
        </button>
      </div>
    </header>
  );
}

export const color = {
  ink: '#111111',
  paper: '#F2F0EC',
  canvas: '#DEDBD4',
  white: '#FFFFFF',
  signal: '#E2231A',
  signalTint: '#FFD9D6',
  muted: '#7D786E',
  muted2: '#8A857C',
  bodyDim: '#4F4A42',
  fieldText: '#6B665E',
  ruleLight: '#DDDDDD',
  ruleGrid: '#E0DCD4',
  ruleDark: '#3A3733',
  adminHead: '#1C1A18',
  textDimDark: '#CFCABF',
  textDimDark2: '#A8A299',
  inactive: '#DCD8D0',
  tableHead: '#E6E3DC',
  cellBlank: '#ECEAE4',
} as const;

export const font = {
  ui: "'Archivo', sans-serif",
  alt: "'Space Grotesk', sans-serif",
  mono: "'JetBrains Mono', monospace",
} as const;

export const fontSize = {
  hero: 60, section: 42,
  titleXl: 38, titleL: 34, titleM: 30, titleS: 26,
  titleMobile: 22, titleMobileS: 20,
  numberL: 26, numberM: 22,
  card: 15, cardMobile: 12.5,
  value: 12, body: 12.5, bodyS: 11.5,
  caption: 9, label: 10, kicker: 11,
  chip: 9, chipS: 8,
  button: 11, buttonS: 9, nav: 9,
} as const;

export const tracking = {
  tight: '-0.03em', tight2: '-0.02em',
  button: '0.07em', chip: '0.12em', caption: '0.13em',
  label: '0.14em', kicker: '0.18em',
} as const;

export const lineHeight = {
  display: 0.94, card: 1.02, tight: 1.1, value: 1.25, body: 1.45,
} as const;

export const space = [2, 4, 5, 6, 8, 10, 12, 14, 16, 18, 20, 26, 40, 48, 56] as const;

export const geometry = {
  radius: 0,
  rule: 1,
  ruleStrong: 2,
  signalBar: 4,
  maxWidth: 1360,
  gutter: 48,
  gutterNarrow: 20,
  transition: '120ms linear',
  minTapTarget: 44,
} as const;

/** Categories are transmitted as numerals, never colours. */
export const categories = {
  '01': 'Фестивали',
  '02': 'Медиации',
  '03': 'Лекции',
  '04': 'Кино',
  '05': 'Спектакли',
  '06': 'Концерты',
} as const;

export type ChipVariant = 'default' | 'active' | 'signal' | 'darkActive' | 'darkMuted';

/** Status → chip variant. Red means "needs attention", never decoration. */
export const statusVariant: Record<string, ChipVariant> = {
  'Опубликовано': 'active',
  'Подтверждено': 'active',
  'Верифицирован': 'active',
  'Черновик': 'default',
  'Прошедшее': 'default',
  'На модерации': 'signal',
  'Модерация': 'signal',
  'Ожидает': 'signal',
  'На проверке': 'signal',
  'Тестовый': 'signal',
};

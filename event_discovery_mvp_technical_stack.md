# Техническая составляющая первой версии приложения для discovery событий

**Проект:** web-first приложение с последующим iOS-приложением, похожее по идее на Luma / городскую афишу / marketplace событий.  
**Регион запуска:** Россия, один город на старте, например Москва.  
**Ориентир нагрузки MVP:** до 10 000 посещений в день, пиково к выходным.  
**Дата актуализации:** 2026-05-24.  

---

## 1. Резюме рекомендации

Для первой версии я рекомендую не строить полноценную микросервисную систему. Оптимальная архитектура:

```text
Web / iOS client
      |
      v
API Gateway / Load Balancer
      |
      v
Go backend: modular monolith
      |
      +--> PostgreSQL + PostGIS
      +--> Redis
      +--> S3 object storage
      +--> Email provider
      +--> APNs / Push provider
      +--> Maps / Geocoding API
      +--> AI provider adapter
      +--> Analytics / Monitoring
```

Главный принцип: **один backend на Go, но с чётко разделёнными доменными модулями**. Это даст простоту разработки, деплоя и отладки. Когда появится реальная нагрузка или отдельная команда, можно вынести в отдельные сервисы: `notifications`, `search`, `recommendations`, `ai-assistant`, `payments`.

---

## 2. Целевые требования MVP

### 2.1. Пользовательская часть

- Просмотр событий в городе.
- Фильтры: дата, категория, интересы, район/локация, цена, формат.
- Поиск по названию, описанию, месту и организатору.
- Сохранение событий в избранное.
- Регистрация / RSVP на событие.
- Добавление события в календарь через `.ics`.
- Email-напоминания.
- Push-напоминания после появления iOS-приложения.
- Простая AI-подборка событий по текстовому запросу.

### 2.2. Организаторская часть

- Регистрация организатора.
- Профиль организатора.
- Создание события.
- Редактирование события.
- Загрузка обложки.
- Указание места, даты, времени, цены, тегов, категории.
- Статус публикации: draft / pending_review / published / rejected / cancelled.
- Просмотр базовой статистики: просмотры, сохранения, регистрации.

### 2.3. Админская часть

- Модерация организаторов.
- Модерация событий.
- Жалобы на события.
- Ручное продвижение событий.
- Управление категориями и интересами.
- Базовая аналитика.

---

## 3. Рекомендуемый стек первой версии

## 3.1. Frontend web

**Рекомендация:** Next.js + TypeScript.

Почему:

- Хорошо подходит для SEO-страниц событий.
- Можно делать server-side rendering для публичных event pages.
- Удобно использовать один стек для landing, каталога, профиля пользователя и кабинета организатора.
- Большая экосистема UI-библиотек.

**Библиотеки:**

- Next.js.
- TypeScript.
- Tailwind CSS.
- TanStack Query для работы с API.
- Zod для валидации форм.
- React Hook Form.
- shadcn/ui или Radix UI для базовых компонентов.

**Что не брать на старте:**

- Сложный frontend microfrontend.
- GraphQL только ради моды.
- Собственную дизайн-систему с нуля.

---

## 3.2. iOS-приложение

**Рекомендация для первой мобильной версии:** SwiftUI, если есть iOS-разработчик; React Native / Expo, если команда больше web-oriented.

### Вариант A: SwiftUI

Плюсы:

- Нативные push, calendar, geolocation, sharing.
- Лучше воспринимается App Store review.
- Проще сделать качественный iOS UX.

Минусы:

- Нужен iOS-разработчик.
- Логика клиента частично дублируется с web.

### Вариант B: React Native / Expo

Плюсы:

- Быстрее для web-команды.
- Можно частично переиспользовать TypeScript-модели и UI-подходы.
- Хорошо для MVP.

Минусы:

- Нативные интеграции иногда сложнее.
- Больше риска технического долга.

### Что я бы сделал

1. Сначала web MVP.
2. Потом iOS-приложение как нативный клиент к тому же API.
3. Не делать просто WebView-приложение, если цель — полноценный App Store продукт.

---

## 3.3. Backend

**Рекомендация:** Go backend как modular monolith.

### Фреймворк / router

Подойдут:

- `chi` — минималистично, удобно, production-friendly.
- `gin` — быстрее старт, много примеров.
- `echo` — тоже нормальный вариант.

**Моя рекомендация:** `chi` + стандартная библиотека Go, если хочется аккуратной архитектуры без лишней магии.

### Доменные модули

```text
/internal/users
/internal/auth
/internal/organizers
/internal/events
/internal/venues
/internal/categories
/internal/interests
/internal/search
/internal/recommendations
/internal/rsvp
/internal/calendar
/internal/notifications
/internal/moderation
/internal/admin
/internal/ads
/internal/ai
/internal/billing
```

### API

**Рекомендация для MVP:** REST API.

Почему REST:

- Проще поддерживать.
- Хорошо документируется через OpenAPI.
- Удобно для web и iOS.
- Меньше инфраструктурной сложности, чем GraphQL.

**Документация API:**

- OpenAPI 3.0.
- Swagger UI только для internal/staging.

### Что не брать на старте

- gRPC между внутренними сервисами.
- Kafka.
- Service mesh.
- Kubernetes-first подход.
- Полноценная микросервисная архитектура.

---

## 3.4. База данных

**Рекомендация:** PostgreSQL + PostGIS.

Почему:

- События имеют географию: город, координаты, расстояние, район, venue.
- PostgreSQL хорошо справится с MVP-нагрузкой.
- PostGIS позволит делать geo-запросы: события рядом, расстояние до места, фильтрация по области.
- Можно начать с PostgreSQL full-text search и trigram search без отдельного search engine.

### Основные таблицы

```text
users
user_profiles
user_interests
organizers
organizer_members
events
event_dates
event_locations
event_categories
event_tags
event_images
event_status_history
event_rsvps
saved_events
notification_jobs
notification_deliveries
moderation_queue
complaints
admin_actions
ai_query_logs
```

### Важные индексы

- `events(status, city_id, starts_at)`.
- `events(organizer_id)`.
- `event_dates(starts_at, ends_at)`.
- `event_locations USING GIST(coordinates)`.
- Full-text index по `title`, `description`, `venue_name`.
- Trigram index для fuzzy search.

### Бэкапы

Минимум:

- ежедневные backup snapshots;
- хранение 7 ежедневных и 4 недельных копий;
- ежемесячный тест восстановления;
- отдельная backup bucket/prefix в S3;
- доступ к backup только у backend/DevOps роли.

---

## 3.5. Кэш и фоновые задачи

**Рекомендация:** Redis.

Использовать для:

- rate limiting;
- short-lived sessions / tokens;
- очередей фоновых задач;
- кэширования популярных подборок;
- anti-spam лимитов;
- временных OTP-кодов.

### Очереди

Для Go хорошо подойдут:

- `asynq` поверх Redis;
- или собственный простой job worker через таблицу PostgreSQL + outbox pattern.

**Моя рекомендация:** PostgreSQL outbox для критичных событий + Redis/Asynq для удобной обработки задач.

---

## 3.6. Хранение изображений

**Рекомендация:** S3-compatible object storage.

Хранить:

- обложки событий;
- аватары организаторов;
- изображения venue;
- generated thumbnails.

Не хранить изображения в PostgreSQL.

### Практическая схема

```text
s3://bucket/events/{event_id}/original.jpg
s3://bucket/events/{event_id}/cover_1200.webp
s3://bucket/events/{event_id}/cover_600.webp
s3://bucket/organizers/{organizer_id}/avatar.webp
```

### Обработка изображений

На старте:

- resize и conversion в backend worker;
- WebP как основной формат;
- лимит размера исходного файла;
- проверка MIME type;
- удаление EXIF-метаданных.

Позже:

- отдельный image processing service;
- CDN;
- smart crop.

---

## 3.7. Поиск

### Первая версия

**Рекомендация:** начать с PostgreSQL search.

Использовать:

- full-text search;
- trigram search;
- weighted ranking;
- фильтры по дате, категории, району, цене;
- PostGIS для расстояния.

### Когда добавлять отдельный search engine

Добавлять Meilisearch / OpenSearch, если появятся:

- тысячи и десятки тысяч активных событий;
- сложные typo-tolerant запросы;
- faceted search;
- отдельный индекс рекомендаций;
- высокая нагрузка на поиск.

### Рекомендация по выбору

- **Meilisearch** — проще для MVP, хороший UX поиска.
- **OpenSearch** — мощнее, но тяжелее в эксплуатации.

Для первой версии: **не поднимать OpenSearch без необходимости**.

---

## 4. Рекомендуемая инфраструктура хостинга

## 4.1. Основной рекомендуемый вариант

**Рекомендация:** Selectel или Yandex Cloud.

### Почему Selectel

- Российская инфраструктура.
- Есть облачные серверы, managed databases, S3, load balancer, Kubernetes на будущее.
- Хороший баланс между enterprise-инфраструктурой и понятной ценой.
- Подходит для хранения персональных данных пользователей РФ.

### Почему Yandex Cloud

- Хорошая связка с Yandex Maps, YandexGPT, SmartCaptcha, Monitoring, Object Storage.
- Удобно, если вы планируете использовать много сервисов Яндекса.
- Есть managed PostgreSQL, Object Storage, Compute, Load Balancer.

### Почему Timeweb Cloud можно рассмотреть

- Хорош для дешёвого MVP и staging.
- Есть VDS/VPS, cloud servers, DBaaS, S3, Kubernetes.
- Обычно проще и дешевле для старта, но для более серьёзного production я бы сравнивал с Selectel/Yandex.

### Почему Cloud.ru / VK Cloud можно рассмотреть

- Хороши для корпоративного контура, закупок, enterprise-контрактов.
- Имеют широкий набор cloud-сервисов.
- Для маленького MVP могут быть избыточны по процессу подключения и сопровождения.

---

## 4.2. Production MVP: рекомендуемый набор ресурсов

### Минимально нормальная production-конфигурация

| Компонент | Количество | Рекомендованный размер | Комментарий |
|---|---:|---:|---|
| App VM | 2 | 2 vCPU / 4 GB RAM | Backend + web frontend, Docker containers |
| PostgreSQL | 1 managed cluster | 2 vCPU / 4-8 GB RAM / 50-100 GB SSD | Лучше managed, чем self-hosted |
| Redis | 1 | 1-2 GB RAM | Cache, rate limit, jobs |
| S3 bucket | 1-2 | 50-200 GB на старте | Изображения + backup exports |
| Load Balancer | 1 | базовый | HTTPS termination + routing |
| Public IP | 1-2 | стандарт | Для LB / bastion |
| Monitoring VM | 0-1 | 1 vCPU / 1-2 GB RAM | Можно начать с managed/SaaS |
| Staging VM | 1 | 2 vCPU / 4 GB RAM | Отдельное окружение для тестов |

### Ориентир бюджета

| Сценарий | Стоимость в месяц |
|---|---:|
| Дешёвый beta MVP на VPS | 3 000-15 000 ₽ |
| Нормальный production MVP | 20 000-60 000 ₽ |
| Production MVP с запасом, managed DB, staging, мониторингом | 50 000-100 000 ₽ |
| Kubernetes-first вариант | 60 000-150 000+ ₽ |

**Моя рекомендация на старт:** целиться в **30 000-70 000 ₽/мес** для нормального production MVP, если есть публичный запуск и реальные пользователи.

---

## 4.3. Экономный beta-вариант

Подходит для закрытого теста, 100-1000 пользователей, без сильного marketing push.

| Компонент | Конфигурация |
|---|---|
| VPS #1 | Backend + frontend + PostgreSQL + Redis |
| S3 | Изображения событий |
| Backups | Ежедневный `pg_dump` + S3 |
| Monitoring | Uptime Kuma + Sentry free |
| Email | Mailganer / SendPulse / Unisender |

Плюсы:

- дешево;
- быстро;
- удобно для проверки гипотезы.

Минусы:

- больше ручного администрирования;
- слабее отказоустойчивость;
- PostgreSQL на той же машине — риск для production.

---

## 4.4. Что не нужно в первой версии

- Kubernetes.
- Kafka.
- Service mesh.
- Multi-region deployment.
- OpenSearch cluster.
- Собственный CDN.
- Собственный email server.
- Собственная SMS-инфраструктура.
- Собственная система платежей.

---

## 5. Внешние технологии и сервисы

## 5.1. Email-уведомления

### Зачем нужно

- Подтверждение регистрации.
- Magic link / email OTP.
- Напоминания о событиях.
- Уведомления организаторам о модерации.
- Еженедельная подборка событий.

### Рекомендация

Для транзакционных писем:

1. **Unisender Go** — хороший вариант для российского production-контура.
2. **Mailganer SMTP** — простой и недорогой старт.
3. **SendPulse SMTP** — можно использовать на MVP, есть бесплатный стартовый объём.

Для маркетинговых рассылок:

- Unisender.
- DashaMail.
- SendPulse.

### Настройки, которые обязательны

- SPF.
- DKIM.
- DMARC.
- Отдельный поддомен для отправки, например `notify.example.com`.
- Отдельный поддомен для marketing email, например `mail.example.com`.
- Отписка от маркетинговых писем.
- Раздельные категории: transactional и marketing.

### Что отправлять на старте

| Письмо | Тип | Приоритет |
|---|---|---:|
| Подтверждение email | transactional | P0 |
| Magic link / OTP | transactional | P0 |
| Подтверждение регистрации на событие | transactional | P0 |
| Напоминание за 24 часа | transactional | P1 |
| Напоминание за 2 часа | transactional | P1 |
| Подборка событий на выходные | marketing / digest | P2 |
| Уведомление организатору о статусе модерации | transactional | P0 |

---

## 5.2. Push-уведомления

### Для iOS

Использовать Apple Push Notification service, APNs.

В backend хранить:

```text
user_id
device_token
platform
app_version
locale
timezone
last_seen_at
is_active
```

### Варианты реализации

**Вариант A: напрямую через APNs**

Плюсы:

- меньше третьих сторон;
- лучше контроль над данными;
- дешевле;
- нормально подходит для iOS-only первой версии.

Минусы:

- нужно самостоятельно поддерживать delivery logic;
- нужно обрабатывать invalid tokens.

**Вариант B: Firebase Cloud Messaging**

Плюсы:

- удобно, если позже будет Android и web push;
- есть связка с Firebase Analytics / Crashlytics.

Минусы:

- внешний Google-контур;
- надо оценить требования по персональным данным.

**Моя рекомендация:** для iOS-first продукта начать с **APNs напрямую**, а FCM рассматривать при появлении Android/web push.

---

## 5.3. SMS

### Рекомендация

Не использовать SMS как основной канал на старте. SMS дорогой и усложняет регистрацию.

Использовать SMS только если:

- нужен вход по номеру телефона;
- есть риск фейковых аккаунтов;
- организаторы требуют телефон участника;
- нужна критичная доставка кодов подтверждения.

### Возможные провайдеры

- SMS.ru.
- SMS Aero.
- SMSC.
- Devino Telecom.

### MVP-подход

Сначала:

- email login / magic link;
- optional phone field;
- phone verification только для организаторов или позже.

---

## 5.4. Карты, адреса и геокодинг

### Зачем нужно

- Проверка адреса организатором.
- Показ места события на карте.
- Поиск событий рядом.
- Фильтр по району/метро/расстоянию.

### Рекомендованные варианты

#### Вариант A: Yandex Maps API

Хорошо подходит для России и Москвы.

Использовать:

- JavaScript API для web-карт.
- Geocoder / Geosuggest для ввода адреса.
- MapKit позже для iOS, если нужна нативная карта.

#### Вариант B: 2GIS API

Хорош для городских POI, организаций и точных venue.

Использовать:

- Map Tiles.
- Geocoder.
- Places API.
- Suggest API.

#### Вариант C: DaData

Использовать для:

- подсказок адреса;
- нормализации адресов;
- валидации организаций/ИНН, если организаторы будут юридическими лицами.

### Моя рекомендация для первой версии

- Для публичной карты и геокодинга: **Yandex Maps API** или **2GIS**.
- Для нормализации адресов и данных организаторов: **DaData**.
- В базе хранить не только текст адреса, но и координаты.

```text
venue_name
address_raw
address_normalized
city
district
metro
lat
lon
source: manual | yandex | 2gis | dadata
```

---

## 5.5. Календарная интеграция

### MVP

Сделать без OAuth:

- генерация `.ics` файла;
- кнопка «Добавить в Apple Calendar»;
- кнопка «Добавить в Google Calendar» через URL;
- email с `.ics` attachment после RSVP.

### Что отложить

- Google Calendar OAuth sync.
- Apple EventKit sync из backend.
- Двусторонняя синхронизация изменений.

### Почему

Двусторонняя календарная синхронизация сложна: права доступа, revoke токенов, изменение времени событий, удаление, recurring events. Для MVP достаточно `.ics`.

---

## 5.6. Платежи и билеты

### Рекомендация для первой версии

Не делать полноценную продажу билетов в MVP, если это не основная бизнес-модель с первого дня.

Сначала:

- бесплатные RSVP;
- внешняя ссылка на оплату/билет, если организатор продаёт билеты на Timepad/другой платформе;
- ручное поле `external_ticket_url`.

Позже:

- ЮKassa;
- CloudPayments;
- Robokassa;
- Т-Банк эквайринг;
- СБП.

### Почему отложить

Платежи добавляют:

- фискализацию 54-ФЗ;
- возвраты;
- комиссии;
- спорные транзакции;
- split payments / marketplace settlement;
- юридическую модель между платформой, организатором и участником.

Если продавать билеты, надо заранее решить: вы агент, продавец, marketplace, рекламная площадка или просто витрина.

---

## 5.7. AI-помощник

### Что делать в MVP

AI должен быть не «болталкой», а conversational search.

Примеры запросов:

- «Подбери что-нибудь на выходные вечером в Москве до 3000 рублей».
- «Хочу мастер-класс по керамике или искусству, не слишком далеко от центра».
- «Куда пойти одному, чтобы познакомиться с людьми?»

### Архитектура AI

```text
User text query
   |
   v
AI intent parser
   |
   v
Structured search filters
   |
   v
Backend search in real events DB
   |
   v
AI response summarizer
   |
   v
Response with real event IDs only
```

### Важно

AI не должен выдумывать события. Он должен отвечать только на базе событий, которые вернул backend.

### Провайдеры

- GigaChat API.
- YandexGPT / Yandex AI Studio.
- OpenAI / Anthropic только если юридически и платёжно допустимо для проекта.

### Что хранить

- query text;
- detected intent;
- filters;
- returned event IDs;
- model name;
- token usage;
- feedback пользователя.

### Ограничения

- rate limit на пользователя;
- дневной лимит AI-запросов;
- отдельный prompt/version control;
- запрет на генерацию событий вне базы;
- логирование ошибок и пустых выдач.

---

## 5.8. Аналитика продукта

### Web

- Yandex Metrica.
- PostHog self-hosted или cloud, если нужна продуктовая аналитика событий.
- Plausible/Matomo, если нужен более privacy-friendly подход.

### iOS

- AppMetrica.
- Firebase Analytics, если уже используется Firebase.
- Собственная event analytics в PostgreSQL/ClickHouse позже.

### Основные события аналитики

```text
user_signed_up
organizer_signed_up
event_created
event_submitted_for_review
event_published
event_viewed
event_saved
event_shared
event_rsvp_started
event_rsvp_completed
calendar_add_clicked
search_performed
filter_applied
ai_query_submitted
ai_result_clicked
complaint_submitted
```

### KPI MVP

- Количество опубликованных событий.
- Доля событий, прошедших модерацию.
- Количество активных организаторов.
- Event views per user.
- Save rate.
- RSVP conversion.
- Retention W1/W4.
- CTR AI-подборок.
- Доля пользователей, добавивших событие в календарь.

---

## 5.9. Мониторинг и ошибки

### Минимальный набор

- Sentry для backend и frontend ошибок.
- Uptime Kuma или внешний uptime monitoring.
- Prometheus + Grafana для метрик, если есть DevOps-ресурс.
- Loki или Vector + object storage для логов.
- Telegram/Slack alert channel.

### Что мониторить

- API latency p50/p95/p99.
- Error rate.
- PostgreSQL CPU/RAM/storage.
- Slow queries.
- Redis memory.
- Queue depth.
- Email delivery failures.
- Push delivery errors.
- S3 upload failures.
- AI provider errors and costs.

### P0-алерты

- API down.
- PostgreSQL unavailable.
- Ошибки авторизации массово растут.
- Очередь уведомлений не обрабатывается.
- Бэкап не прошёл.
- Диск PostgreSQL > 80%.

---

## 5.10. CAPTCHA и антиспам

### Где нужна защита

- регистрация пользователя;
- регистрация организатора;
- создание события;
- отправка жалоб;
- массовые RSVP;
- AI-запросы.

### Возможные сервисы

- Yandex SmartCaptcha.
- hCaptcha.
- Cloudflare Turnstile.

### Моя рекомендация

Для российского рынка начать с **Yandex SmartCaptcha** или **hCaptcha**. Для организаторов дополнительно использовать ручную модерацию и лимиты.

---

## 6. Auth и безопасность

## 6.1. Авторизация

### Рекомендация для MVP

- Email magic link.
- Email OTP.
- Password login как опция позже.
- OAuth через Apple ID для iOS после выхода приложения.

### Почему не phone-first

- SMS стоит денег.
- SMS усложняет onboarding.
- Для discovery-приложения email обычно достаточно.

### Хранить

```text
users.email
users.email_verified_at
users.phone_optional
users.role
sessions
refresh_tokens
login_attempts
audit_log
```

---

## 6.2. Роли доступа

Минимальные роли:

```text
anonymous
user
organizer_member
organizer_admin
moderator
admin
super_admin
```

Для организаторов лучше сразу делать membership-модель, потому что у одного организатора может быть несколько сотрудников.

---

## 6.3. Security checklist

- HTTPS везде.
- HSTS.
- Rate limiting на auth endpoints.
- CSRF protection, если cookie-based auth.
- Secure, HttpOnly cookies.
- Argon2id, если появятся пароли.
- Audit log для admin actions.
- Signed URLs для загрузки в S3.
- Проверка MIME type изображений.
- Antivirus scan later, если будет массовая загрузка файлов.
- Secrets не хранить в git.
- Separate production/staging credentials.

---

## 7. Юридическая и privacy часть

Для российского рынка с пользователями из РФ сразу подготовить:

- Политика обработки персональных данных.
- Пользовательское соглашение.
- Согласие на обработку персональных данных.
- Согласие на маркетинговые рассылки отдельно от transactional.
- Cookie notice.
- Механизм удаления аккаунта.
- Экспорт/удаление персональных данных по запросу.
- Уведомление Роскомнадзора как оператор ПДн, если применимо.
- Хранение первичных баз данных пользователей РФ в РФ.

### Данные, которые лучше минимизировать

- Не собирать точную геолокацию без необходимости.
- Не требовать телефон на старте.
- Не хранить лишние поля о пользователе.
- Интересы и историю кликов хранить с понятной целью.

---

## 8. DevOps и CI/CD

## 8.1. Репозитории

Вариант 1: monorepo.

```text
/apps/web
/apps/api
/apps/admin
/packages/shared
/infra
/docs
```

Вариант 2: отдельные репозитории.

Для MVP я бы выбрал **monorepo**, если команда маленькая.

## 8.2. CI/CD

Подойдут:

- GitHub Actions.
- GitLab CI.
- Gitea Actions, если нужен self-hosted контур.

### Pipeline

```text
lint
unit tests
integration tests
build docker image
push image to registry
run migrations
blue/green or rolling deploy
smoke tests
```

## 8.3. Container registry

- GitHub Container Registry.
- GitLab Registry.
- Selectel Container Registry.
- Yandex Container Registry.

Для российского production-контура лучше использовать registry внутри того же облака, если есть требование к локализации/доступности.

---

## 9. Среды

## 9.1. Local

- Docker Compose.
- PostgreSQL.
- Redis.
- MinIO как S3 emulator.
- Mailpit для email.

## 9.2. Staging

- отдельная VM;
- отдельная PostgreSQL database или отдельный маленький managed cluster;
- отдельный S3 bucket;
- fake payments;
- test email domain;
- seed data.

## 9.3. Production

- 2 app instances;
- managed PostgreSQL;
- Redis;
- S3;
- LB;
- backups;
- monitoring;
- alerts.

---

## 10. Пример первой production-сметы

Ниже не бухгалтерская смета, а технический ориентир.

| Категория | Минимум | Нормально | Комментарий |
|---|---:|---:|---|
| App servers | 3 000-8 000 ₽ | 8 000-20 000 ₽ | 1-2 VM |
| Managed PostgreSQL | 4 000-15 000 ₽ | 15 000-40 000 ₽ | Зависит от HA и storage |
| Redis | 500-3 000 ₽ | 3 000-10 000 ₽ | Можно self-hosted на старте |
| S3 + трафик | 500-3 000 ₽ | 3 000-15 000 ₽ | Зависит от картинок и CDN |
| Load Balancer / IP / DNS | 1 000-5 000 ₽ | 3 000-10 000 ₽ | LB + public IP |
| Monitoring / logs | 0-5 000 ₽ | 5 000-20 000 ₽ | Sentry/Grafana/logs |
| Email | 0-2 000 ₽ | 2 000-10 000 ₽ | Зависит от объёма |
| Maps/geocoding | 0-10 000 ₽ | 10 000-50 000 ₽ | Может стать заметной статьёй |
| AI | 0-5 000 ₽ | 5 000-30 000 ₽ | Лимитировать usage |
| SMS | 0 ₽ | 1 000-20 000 ₽ | Лучше не использовать массово |

**Реалистичный бюджет первой production-версии:** 30 000-70 000 ₽/мес.  
**Бюджет с хорошим запасом и несколькими managed-сервисами:** 70 000-120 000 ₽/мес.

---

## 11. Рекомендуемый список сервисов по категориям

| Категория | Рекомендация | Альтернативы | Брать сразу? |
|---|---|---|---|
| Cloud hosting | Selectel | Yandex Cloud, Timeweb Cloud, Cloud.ru, VK Cloud | Да |
| Compute | 2 VM | 1 VPS для beta | Да |
| Database | Managed PostgreSQL + PostGIS | Self-hosted PostgreSQL для beta | Да |
| Cache / jobs | Redis | PostgreSQL queue only | Да |
| Object storage | S3-compatible | MinIO self-hosted только local/dev | Да |
| Search | PostgreSQL FTS | Meilisearch, OpenSearch | PostgreSQL сразу, Meilisearch позже |
| Email transactional | Unisender Go / Mailganer | SendPulse, DashaMail | Да |
| Marketing email | Unisender / DashaMail | SendPulse | Позже |
| Push iOS | APNs direct | FCM, Pushwoosh, OneSignal | С iOS-версией |
| SMS | SMS.ru / SMS Aero / SMSC | Devino Telecom | Не сразу |
| Maps | Yandex Maps API | 2GIS, DaData | Да, но с лимитами |
| Address suggestions | DaData | Yandex Geosuggest, 2GIS Suggest | Желательно |
| Calendar | `.ics` generation | Google Calendar OAuth | Да, без OAuth |
| Payments | External ticket URL | YooKassa, CloudPayments, Robokassa | Не сразу |
| AI | GigaChat / YandexGPT | OpenAI, Anthropic | P1/P2, с лимитами |
| Analytics web | Yandex Metrica | PostHog, Matomo | Да |
| Analytics mobile | AppMetrica | Firebase Analytics | С iOS-версией |
| Error tracking | Sentry | Firebase Crashlytics, self-hosted Sentry | Да |
| Monitoring | Uptime Kuma + Grafana | Yandex Monitoring, Cloud.ru Monitoring | Да |
| CAPTCHA | Yandex SmartCaptcha | hCaptcha, Turnstile | Да для public forms |
| Admin panel | Own admin UI | Appsmith, Retool-like tools | Да, минимально |

---

## 12. Что реализовать в первой версии по приоритетам

## P0: обязательно

- Backend Go.
- PostgreSQL + PostGIS.
- Event CRUD.
- Organizer account.
- User account.
- Admin moderation.
- Search/filter by date/category/location.
- S3 image uploads.
- Email notifications.
- `.ics` calendar export.
- Basic analytics.
- Sentry/error monitoring.
- Backups.
- Legal pages.

## P1: очень желательно

- Push notifications после iOS-релиза.
- AI-подборка событий.
- DaData/Yandex/2GIS address suggestions.
- Saved events.
- Organizer analytics.
- Complaint system.
- Featured events.
- Basic anti-spam scoring.

## P2: позже

- Payments and ticketing.
- Paid promotion self-service.
- Meilisearch/OpenSearch.
- Recommendation ML.
- Multi-city.
- Public organizer API.
- Calendar OAuth sync.
- Chat between users/organizers.
- Full microservices.
- Kubernetes.

---

## 13. Минимальная схема доменной модели

```text
User
- id
- email
- name
- role
- created_at

Organizer
- id
- name
- description
- website_url
- verification_status
- created_at

OrganizerMember
- organizer_id
- user_id
- role

Event
- id
- organizer_id
- title
- description
- status
- city_id
- venue_id
- price_type
- price_min
- price_max
- external_ticket_url
- starts_at
- ends_at
- published_at
- created_at

Venue
- id
- name
- address
- lat
- lon
- metro
- district

RSVP
- id
- event_id
- user_id
- status
- created_at

SavedEvent
- user_id
- event_id
- created_at
```

---

## 14. Первые технические решения, которые лучше зафиксировать заранее

1. **Храним персональные данные в российском облаке.**
2. **Backend — Go modular monolith.**
3. **PostgreSQL + PostGIS как главный источник истины.**
4. **S3 для изображений, не база данных.**
5. **События проходят модерацию до публикации.**
6. **AI не генерирует события, а только помогает искать по базе.**
7. **На старте нет встроенной продажи билетов.**
8. **Календарь через `.ics`, без OAuth.**
9. **Email обязателен; SMS не обязателен.**
10. **Kubernetes не нужен в первой версии.**

---

## 15. Источники для проверки цен и сервисов

- Selectel prices: https://selectel.ru/prices/
- Selectel price calculator: https://selectel.ru/prices/calculator/
- Yandex Cloud Compute pricing: https://yandex.cloud/ru/docs/compute/pricing
- Yandex Managed PostgreSQL pricing: https://yandex.cloud/en/docs/managed-postgresql/pricing
- Yandex Object Storage pricing: https://yandex.cloud/en/docs/storage/pricing
- Cloud.ru calculator: https://cloud.ru/calculator
- Timeweb Cloud: https://timeweb.cloud/
- Timeweb Cloud prices: https://timeweb.cloud/prices
- Unisender Go: https://go.unisender.ru/
- Mailganer SMTP pricing: https://mailganer.com/ru/price/smtp
- SendPulse SMTP pricing: https://sendpulse.com/ru/pricing/smtp
- Yandex Maps API tariffs: https://yandex.ru/maps-api/tariffs
- 2GIS API pricing: https://docs.2gis.com/platform-manager/subscription/pricing
- DaData pricing: https://dadata.ru/pricing/
- GigaChat tariffs: https://developers.sber.ru/docs/ru/gigachat/tariffs/individual-tariffs
- Yandex AI Studio pricing: https://aistudio.yandex.ru/docs/en/ai-studio/pricing/
- AppMetrica pricing: https://appmetrica.yandex.com/about/pricing
- Sentry pricing: https://sentry.io/pricing/
- Firebase pricing: https://firebase.google.com/pricing
- Apple Developer Program: https://developer.apple.com/support/compare-memberships/
- Apple APNs documentation: https://developer.apple.com/documentation/usernotifications/registering-your-app-with-apns
- Yandex SmartCaptcha pricing: https://yandex.cloud/en/docs/smartcaptcha/pricing
- SMS.ru pricing: https://sms.ru/price
- SMS Aero pricing: https://smsaero.ru/price/tariff/
- YooKassa API documentation: https://yookassa.ru/developers/
- CloudPayments: https://cloudpayments.ru/

---

## 16. Финальная рекомендация

Для первой версии я бы выбрал следующую сборку:

```text
Hosting: Selectel или Yandex Cloud
Backend: Go + chi
Frontend: Next.js + TypeScript
Database: Managed PostgreSQL + PostGIS
Cache/jobs: Redis + Asynq или PostgreSQL outbox
Storage: S3-compatible object storage
Search: PostgreSQL FTS + PostGIS
Email: Unisender Go или Mailganer
Push: APNs direct после iOS-релиза
Maps: Yandex Maps API или 2GIS
Address normalization: DaData
Calendar: .ics export
AI: GigaChat или YandexGPT, только как search assistant
Analytics: Yandex Metrica + AppMetrica
Errors: Sentry
Monitoring: Uptime Kuma + Grafana/Prometheus
CAPTCHA: Yandex SmartCaptcha
Payments: external_ticket_url на MVP, встроенные платежи позже
```

Такой стек позволит быстро выпустить MVP, не переплачивать за инфраструктуру и не заблокировать будущий рост.

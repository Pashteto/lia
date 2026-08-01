-- Real-venue seed, batch 2: 50 popular Moscow/SPB venues missing from prod
-- (docs/research/venues_seed_2.json). Sourced from the QA finding that
-- "Дом Радио", "ТОН-Центр" and "Noôdome" were unfindable; coordinates come
-- from the Yandex Geocoder (all results precision=exact except two 'number').
--
-- Idempotent: WHERE NOT EXISTS on lower(name) (venues.name has no unique
-- constraint). ids are uuid5(uuid5(URL,'https://presence.tarski.ru/venues'),
-- lower(name)) so a re-run resolves to the SAME id and still no-ops.
-- geog is a generated column from lat/lon — never inserted directly.

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a1ca1217-6088-5721-bb44-8632f0e3de6c', 'Дом Радио', 'Итальянская улица, 27', 'Гостиный двор', '', 59.93562, 30.338542
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дом Радио'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '13bfc88a-c75c-5e3f-8978-ea9a4a5991e3', 'Санкт-Петербургская филармония им. Д.Д. Шостаковича (Большой зал)', 'Михайловская улица, 2', 'Невский проспект', '', 59.936039, 30.331714
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Санкт-Петербургская филармония им. Д.Д. Шостаковича (Большой зал)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fc78c308-9ab5-54ac-8c3c-0d76a9712547', 'Малый зал филармонии им. М.И. Глинки', 'Невский проспект, 30', 'Невский проспект', '', 59.935543, 30.327349
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Малый зал филармонии им. М.И. Глинки'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '09e46839-c680-5a99-9ae3-2604611298cb', 'Санкт-Петербургский Дом музыки', 'набережная реки Мойки, 122', 'Адмиралтейская', '', 59.927924, 30.283592
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Санкт-Петербургский Дом музыки'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '39a89816-115b-5539-8a11-b0c693fe68fa', 'Александринский театр', 'площадь Островского, 6', 'Гостиный двор', '', 59.931774, 30.336278
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Александринский театр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd38553a0-7fa3-5ea3-b69c-ef9d5ae6d7e7', 'Новая сцена Александринского театра', 'набережная реки Фонтанки, 49А', 'Гостиный двор', '', 59.930412, 30.33758
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Новая сцена Александринского театра'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '1fbb0170-e4e4-5187-8b7e-1d2b39b12a69', 'Театр имени Ленсовета', 'Владимирский проспект, 12', 'Достоевская', '', 59.930056, 30.348189
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр имени Ленсовета'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0bf4757c-95e4-5c0c-b033-07a65feddbb0', 'Театр «Мастерская»', 'Народная улица, 1', 'Ломоносовская', '', 59.877786, 30.459158
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр «Мастерская»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b39283f4-8888-527e-82b5-fa82ec40d162', 'ЦВЗ «Манеж»', 'Исаакиевская площадь, 1', 'Адмиралтейская', '', 59.934006, 30.302438
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ЦВЗ «Манеж»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '394ce5b1-f10e-5af7-96ae-527670ee04f3', 'Музей Фаберже', 'набережная реки Фонтанки, 21', 'Гостиный двор', '', 59.934871, 30.343006
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей Фаберже'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '06245212-7746-5dbd-a3aa-b9108866768c', 'Музей Анны Ахматовой в Фонтанном доме', 'Литейный проспект, 53', 'Маяковская', '', 59.936332, 30.347713
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей Анны Ахматовой в Фонтанном доме'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '7719fae0-d9f9-5d7c-aacc-b725fe1fd487', 'Библиотека имени В.В. Маяковского', 'набережная реки Фонтанки, 46', 'Гостиный двор', '', 59.931806, 30.343294
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Библиотека имени В.В. Маяковского'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0d026e0c-2ee4-5fdd-957d-8c6f80831eab', 'Российская национальная библиотека', 'Садовая улица, 18', 'Гостиный двор', '', 59.933406, 30.334661
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Российская национальная библиотека'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ef21990e-84a3-5f77-85e4-76c66cc45b26', 'Бертгольд Центр', 'Гражданская улица, 13-15', 'Садовая', '', 59.928158, 30.312481
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Бертгольд Центр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'aa109acd-e974-5a8a-9ab1-706bb478f951', 'Голицын Лофт', 'набережная реки Фонтанки, 20', 'Гостиный двор', '', 59.940623, 30.341192
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Голицын Лофт'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b0864053-ecc0-5a8b-8883-bb15d4516600', 'Пространство «Скороход»', 'Московский проспект, 107, корпус 5', 'Московские ворота', '', 59.890938, 30.316928
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Пространство «Скороход»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c10c895f-d2b9-549d-bab9-0ac55fd1802a', 'Музей стрит-арта', 'шоссе Революции, 84', 'Ладожская', '', 59.961251, 30.452789
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей стрит-арта'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9da9fce7-4908-550d-819f-6c671ddec127', 'Планетарий №1', 'набережная Обводного канала, 74, литера Ц', 'Балтийская', '', 59.911624, 30.331095
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Планетарий №1'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0a98e727-0a33-5b9e-abb6-f82c305ab973', 'Анненкирхе', 'Кирочная улица, 8В', 'Чернышевская', '', 59.94476, 30.35207
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Анненкирхе'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '4824e298-c130-5af4-a836-ec0adc2a4071', 'Яани Кирик', 'улица Декабристов, 54А', 'Садовая', '', 59.924014, 30.286673
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Яани Кирик'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c4e4a1c0-1810-5843-b365-07ba30073e6c', 'Клуб «Космонавт»', 'Бронницкая улица, 24', 'Технологический институт', '', 59.913952, 30.323737
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Клуб «Космонавт»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'be649aba-d324-508a-95e0-6d1e36bd2364', 'A2 Green Concert', 'проспект Медиков, 3', 'Петроградская', '', 59.969241, 30.31647
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('A2 Green Concert'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '8fc309f1-96b1-55bd-aa3a-f04201eac8fc', 'Aurora Concert Hall', 'Пироговская набережная, 5/2', 'Площадь Ленина', '', 59.956994, 30.341353
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Aurora Concert Hall'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ea8ef851-72c2-5c79-9231-7ea89473a4f0', 'Клуб «Грибоедов»', 'Воронежская улица, 2А', 'Лиговский проспект', '', 59.919567, 30.350705
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Клуб «Грибоедов»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f67fa419-0a8b-5cdc-933f-09d6d6824b4a', 'Киностудия «Ленфильм»', 'Каменноостровский проспект, 10', 'Горьковская', '', 59.958278, 30.316991
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Киностудия «Ленфильм»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9feb2ccf-d438-555b-a9ad-78d33d00d55e', 'ТОН-Центр', 'Комсомольская площадь, 3/30, строение 1', 'Комсомольская', '', 55.781129, 37.653433
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ТОН-Центр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd614ef54-dfae-5b40-8b09-3f10e3d1a8ac', 'Noôdome', 'Романов переулок, 2, строение 1', 'Библиотека имени Ленина', '', 55.754157, 37.608643
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Noôdome'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a65b3255-7a06-5ff7-b8b5-9e44268fa46e', 'Московская консерватория (Большой зал)', 'Большая Никитская улица, 13/6', 'Охотный Ряд', '', 55.756356, 37.604942
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московская консерватория (Большой зал)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'dc371667-3167-545d-b7ac-4dd3744b010d', 'Концертный зал имени П.И. Чайковского', 'Триумфальная площадь, 4/31', 'Маяковская', '', 55.76877, 37.59595
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Концертный зал имени П.И. Чайковского'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b31c409e-5dca-5fc5-bc95-fc86ee4eb2be', 'Московский международный Дом музыки', 'Космодамианская набережная, 52, строение 8', 'Павелецкая', '', 55.733249, 37.646606
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский международный Дом музыки'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '4ab0248e-d19b-59d8-9d33-0095b46e0b53', 'Концертный зал «Зарядье»', 'улица Варварка, 6, строение 1', 'Китай-город', '', 55.75167, 37.629053
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Концертный зал «Зарядье»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '165b1a27-bafe-5b6d-8e60-37a98e953f89', 'Театр Наций', 'Петровский переулок, 3', 'Пушкинская', '', 55.765944, 37.612748
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр Наций'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c51616c7-94e5-590f-939d-0291acd62ed8', 'Театр «Практика»', 'Большой Козихинский переулок, 30', 'Маяковская', '', 55.765493, 37.594261
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр «Практика»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f2638c73-ab42-53b3-8271-20803f91ef6c', 'Мастерская Петра Фоменко', 'набережная Тараса Шевченко, 29', 'Кутузовская', '', 55.743461, 37.535394
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Мастерская Петра Фоменко'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '3990f3cd-b946-5eff-9ab6-1aba03723212', 'Театр на Таганке', 'улица Земляной Вал, 76/21', 'Таганская', '', 55.743183, 37.653873
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр на Таганке'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ea7879b2-c396-5374-9afe-4f1549799616', 'Ленком Марка Захарова', 'улица Малая Дмитровка, 6', 'Пушкинская', '', 55.767762, 37.606909
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Ленком Марка Захарова'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a2c36a72-1411-51e9-acfd-80cdfdc47c23', 'Театр «Современник»', 'Чистопрудный бульвар, 19А', 'Чистые пруды', '', 55.761781, 37.645941
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Театр «Современник»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'e37e8358-0b3c-58ed-a179-3411632ce4a4', 'Мультимедиа Арт Музей (МАММ)', 'улица Остоженка, 16', 'Кропоткинская', '', 55.741662, 37.598789
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Мультимедиа Арт Музей (МАММ)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'e5f0d80a-e4c8-53bf-933b-eca719c1358e', 'Московский музей современного искусства (MMOMA)', 'улица Петровка, 25', 'Чеховская', '', 55.766962, 37.614285
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский музей современного искусства (MMOMA)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f8dde3c2-7edd-571a-a9d5-98e0f5bca62c', 'Центр Вознесенского', 'улица Большая Ордынка, 46, строение 3', 'Добрынинская', '', 55.73556, 37.624013
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Центр Вознесенского'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c694ef73-81ff-5ae9-bcd2-d9886df385cb', 'Cube.Moscow', 'улица Тверская, 3', 'Охотный Ряд', '', 55.757247, 37.612856
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Cube.Moscow'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6b6f0b82-c06e-50fa-8c9c-5e848c74fdd4', 'Библиотека имени Ф.М. Достоевского', 'Чистопрудный бульвар, 23, строение 1', 'Чистые пруды', '', 55.760276, 37.646813
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Библиотека имени Ф.М. Достоевского'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'eebfdc9e-280c-5e2e-8b2b-48bc0bf65e0c', 'Библиотека иностранной литературы («Иностранка»)', 'Николоямская улица, 1', 'Таганская', '', 55.74824, 37.648142
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Библиотека иностранной литературы («Иностранка»)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c20f9e26-2614-5c3f-8946-d3e514335895', 'Лекторий «Прямая речь»', 'Ермолаевский переулок, 25', 'Маяковская', '', 55.766258, 37.593929
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Лекторий «Прямая речь»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '3548f58a-49ea-5993-984e-eaf28f233b40', 'Депо. Лесная', 'улица Лесная, 20, строение 3', 'Белорусская', '', 55.780218, 37.593084
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Депо. Лесная'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a3d2ff80-2624-5012-b03f-151e4413eb80', 'Аптекарский огород', 'проспект Мира, 26, строение 1', 'Проспект Мира', '', 55.77798, 37.633158
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Аптекарский огород'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b10c23b4-6d2c-5096-b4d1-8e5bd3a30a71', 'Московский планетарий', 'Садовая-Кудринская улица, 5, строение 1', 'Баррикадная', '', 55.761411, 37.583661
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский планетарий'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'dda859c5-e9f1-5268-ad54-e315a9630f5c', 'Кинотеатр «Художественный»', 'Арбатская площадь, 14', 'Арбатская', '', 55.752369, 37.602139
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Кинотеатр «Художественный»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6559e81d-a5a6-5f35-9daf-bf1e3b7044f5', 'Клуб «16 Тонн»', 'Пресненский Вал, 6, строение 1', 'Улица 1905 года', '', 55.764313, 37.564383
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Клуб «16 Тонн»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'afa79328-864d-5364-85f9-f9494e9e20c7', 'Центр современного искусства «Марс»', 'Пушкарёв переулок, 5', 'Сухаревская', '', 55.768967, 37.626771
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Центр современного искусства «Марс»'));

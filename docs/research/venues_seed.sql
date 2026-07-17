-- Real-venue seed: 114 verified Moscow/SPB venues (docs/research/venues_seed.json).

-- Idempotent: safe to re-run (WHERE NOT EXISTS on lower(name), since venues.name

-- has no unique constraint). ids are deterministic (uuid5 of the venue name) so a

-- second run resolves to the SAME id and still no-ops via the NOT EXISTS guard.

-- geog is a generated column from lat/lon — never inserted directly.



INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '4bc6a7ea-b50f-5c43-9e32-b73acbf52a13', 'Московский буддийский центр Алмазного пути', 'Каланчёвский тупик, дом 3-5, строение 6', 'Комсомольская', '', 55.771944, 37.653474
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский буддийский центр Алмазного пути'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '49256f2e-0ea5-50ac-8217-e6134d11977a', 'Московский Буддийский Центр Ламы Цонкапы', 'Мытная ул., дом 23, корп. 1', 'Тульская', '', 55.778719, 37.673251
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский Буддийский Центр Ламы Цонкапы'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'efa902be-28d6-52a8-ad26-781a2e464970', 'Буддийский центр Дзогчен Шри-Сингха', 'ул. Мосфильмовская, д. 2В', 'Мосфильмовская', '', 55.717255, 37.522528
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Буддийский центр Дзогчен Шри-Сингха'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '7c266d92-04ed-503f-80df-cbac27e9b67a', 'Тибетский Дом в Москве', 'Рождественский бульвар', 'Трубная', '', 55.766425, 37.625968
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Тибетский Дом в Москве'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '355f09fd-d531-5331-a88b-2a1b22e72fcd', 'Московская Соборная мечеть', 'Выползов переулок, 7', 'Проспект Мира', '', 55.778942, 37.626991
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московская Соборная мечеть'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fb7615d7-4f6c-5912-89dd-6c8e2d233db9', 'Московская хоральная синагога', 'Большой Спасоглинищевский переулок, 10', 'Китай-город', 'Басманный', 55.755449, 37.635111
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московская хоральная синагога'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '848c41de-f43b-5681-a7fa-6a5d985c7d13', 'Дацан Гунзэчойнэй', 'Приморский проспект, 91', 'Старая Деревня', '', 59.983499, 30.255972
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дацан Гунзэчойнэй'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '848e0112-0aaa-593c-848a-ab346dd577ea', 'Буддийский центр Алмазного пути традиции Карма Кагью', 'Никольский переулок, 7 (офис 26)', 'Садовая', '', 59.919951, 30.303914
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Буддийский центр Алмазного пути традиции Карма Кагью'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd2492abe-906b-5754-b06e-a253379211af', 'Иоанновский ставропигиальный женский монастырь', 'наб. реки Карповки, 45', 'Петроградская', '', 59.970798, 30.300298
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Иоанновский ставропигиальный женский монастырь'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ebaa126d-6d1d-56b2-b3a4-ebfac173bd7b', 'Духовно-просветительский центр Санкт-Петербургской епархии', 'Невский проспект, 177', 'Площадь Александра Невского', '', 59.92371, 30.384167
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Духовно-просветительский центр Санкт-Петербургской епархии'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '8fa6f828-b43f-54f8-a0c0-deb5a2a57bf0', 'Artplay', 'Нижняя Сыромятническая ул., 10', 'Чкаловская', 'Басманный', 55.752334, 37.671329
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Artplay'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9fdd8b7c-1c76-58a9-97b6-bca053e53b9f', 'Дизайн-завод «Флакон»', 'ул. Большая Новодмитровская, 36, стр. 12', 'Дмитровская', 'Бутырский', 55.805017, 37.585533
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дизайн-завод «Флакон»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f9bfc95f-78b1-5441-8bc3-c21943facb26', 'Хлебозавод №9', 'ул. Новодмитровская, 1', 'Дмитровская', 'Бутырский', 55.806632, 37.585691
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Хлебозавод №9'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0f4428d9-25da-5ae2-97d0-f8d19da37ec5', 'ДК «Рассвет»', 'Столярный переулок, 3, корп. 15', 'Улица 1905 года', 'Пресненский', 55.764334, 37.567965
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ДК «Рассвет»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'e83b4ed4-1c13-5d28-85f6-922bf331be14', 'Севкабель Порт', 'Кожевенная линия, 40', '', 'Василеостровский', 59.924329, 30.241901
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Севкабель Порт'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'bf4b64a2-04a8-5bb1-a647-1295d3cfb695', 'Лофт Порт (Севкабель Порт)', 'Кожевенная линия, 34', '', 'Василеостровский', 59.923151, 30.243798
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Лофт Порт (Севкабель Порт)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9b45944c-5373-5477-93ea-d5191e48ec51', 'Культурное пространство «Третье место»', 'Литейный проспект, 62', '', 'Центральный', 59.933507, 30.34838
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Культурное пространство «Третье место»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '60b4e207-e0a5-57a0-9f6b-5c7e2e2d6b98', 'Рабочая Станция Plaza', 'ул. Бутырская, 62, 7 этаж', 'Дмитровская', 'Бутырский', 55.802788, 37.583751
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Рабочая Станция Plaza'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9fa5cb6f-2f16-532b-93eb-87f0f0f89330', 'Рабочая Станция Парк Горького', 'Ленинский проспект, 30А', 'Ленинский проспект', 'Якиманка', 55.711722, 37.581417
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Рабочая Станция Парк Горького'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9b8abcd3-bae1-5b26-b41c-f02e9d335316', 'Рабочая Станция Artplay', 'ул. Нижняя Сыромятническая, 10, стр. 2, 7 этаж', 'Чкаловская', 'Басманный', 55.752334, 37.671329
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Рабочая Станция Artplay'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '3deedd0d-ebe5-5343-897a-1120ca8d7e42', 'SOK Рыбаков Тауэр', 'Ленинградский проспект, 36, стр. 11', 'Динамо', 'Аэропорт', 55.788299, 37.567838
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('SOK Рыбаков Тауэр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '3ecacece-ba70-5af5-b649-82fffed992aa', 'Коворкинг Boiler', 'ул. Щербаковская, 16', 'Семёновская', 'Соколиная гора', 55.781416, 37.725007
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Коворкинг Boiler'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '7c3d14d9-fa1b-5dac-b0b6-be088992ddc0', 'Атмосфера Известия', 'Тверская ул., 18, корп. 1', 'Пушкинская / Тверская / Чеховская', 'Тверской', 55.766283, 37.603177
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Атмосфера Известия'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '002d9def-cf93-5068-8d91-36888d2873f8', 'Коворкинг «Ясная Поляна»', 'ул. Льва Толстого, 1-3', 'Петроградская', 'Петроградский', 59.964913, 30.31284
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Коворкинг «Ясная Поляна»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd3ec7d01-da35-521b-aee1-b1b7d5b29f50', 'Практик (8-я линия В.О.)', '8-я линия В.О., 25', 'Василеостровская', 'Василеостровский', 59.939682, 30.279899
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Практик (8-я линия В.О.)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '3ce0a0b2-b0fb-5005-bc5b-1bb4b1a86d40', 'Студия «Воздух»', 'ул. Вавилова, д. 81, корп. 1', 'Профсоюзная / Новые Черёмушки', 'Академический', 55.751541, 37.66459
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Студия «Воздух»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b1479ffe-a1eb-5eb0-b27f-a4e10fd43a2a', 'Йога-дом', 'Петровский переулок, 1/30', 'Чеховская', 'Тверской', 55.765177, 37.611646
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Йога-дом'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '21b946f2-107a-52f9-aafa-7ac04c7532c0', 'Qi Center', 'ул. Архитектора Власова, 18', 'Новые Черёмушки', 'Академический', 55.672158, 37.543644
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Qi Center'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'cfa3b4d7-55f3-5d41-92ee-881d7338294d', 'Bestmemories Фотостудия Лофт', 'проспект Мира, 119, стр. 186', 'ВДНХ', 'Останкинский', 55.827134, 37.629837
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Bestmemories Фотостудия Лофт'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6abafd16-3cae-5b62-8b38-177e1f3b374e', 'LOFT 812', 'ул. 1-я Бухвостова, 12/11, корп. 16, 2 этаж', 'Преображенская площадь', 'Преображенское', 55.824805, 37.415616
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('LOFT 812'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '02838f3b-e677-5bd0-98a2-d733fcd22c50', 'Бахрушинъ LOFT', '1-й Рижский переулок, д. 2, стр. 1', '', 'Мещанский', 55.809365, 37.655023
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Бахрушинъ LOFT'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '1768371e-7fe4-579b-af6c-082c686615c3', 'DIAR танцевальная студия', 'проспект КИМа, д. 6, офис 364, 3 этаж', '', 'Василеостровский', 59.953365, 30.244352
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('DIAR танцевальная студия'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '8a258ad9-3ed9-5502-a6f3-1a3b5394d4d7', 'CORE студия горячей йоги', 'Невский проспект, 118', 'Площадь Восстания', 'Центральный', 59.935625, 30.324654
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('CORE студия горячей йоги'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6b39c450-a779-5881-b0f7-9d995d82a62a', 'Y7studio', 'Петровская коса, 6, корп. 1', '', 'Петроградский', 59.96153, 30.246341
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Y7studio'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ad0322c2-e5b7-5413-9769-374c96828bcc', 'Йога Семья', 'Кронверкский проспект, 27Б', 'Горьковская', 'Петроградский', 59.955358, 30.322994
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Йога Семья'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '26400957-3744-5446-9ab0-46e2cafffd95', 'Галерея «Триумф»', 'Ильинка ул., 3/8, стр. 5', 'Китай-город', 'Тверской', 55.75455, 37.623887
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Галерея «Триумф»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '5da008c8-a1b6-5d69-854e-dc2364ae00bd', 'Галерея Gary Tatintsian', 'Крымский Вал ул., 9, стр. 4', 'Октябрьская / Парк Культуры', 'Якиманка', 55.729154, 37.60561
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Галерея Gary Tatintsian'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '77ab293f-0bf6-5ff6-bf02-1f2ce7c2b3fb', 'Галерея pop/off/art', '4-й Сыромятнический переулок, 1/8, стр. 9 (Винзавод)', 'Курская / Чкаловская', 'Басманный', 55.756157, 37.665459
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Галерея pop/off/art'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '21687bc0-edf8-5c8f-bf2d-6731572e795e', 'ЦТИ «Фабрика»', 'Переведеновский переулок, 18', '', 'Басманный', 55.779627, 37.689664
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ЦТИ «Фабрика»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '7fc249d6-62ae-58ef-a5cd-0c1b7cffcadc', 'Музей современного искусства «Эрарта»', '29-я линия В.О., 2', '', 'Василеостровский', 59.932162, 30.251219
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей современного искусства «Эрарта»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b11d0677-3dd8-527b-9b3d-e175bbcef851', 'Люмьер-Холл', 'наб. Обводного канала, 74, лит. Д', '', 'Адмиралтейский', 59.91042, 30.32904
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Люмьер-Холл'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'aaae63ff-3590-53cd-b768-2f3773888d48', 'Галерея Марины Гисич', 'наб. реки Фонтанки, 121', '', 'Центральный', 59.921049, 30.310591
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Галерея Марины Гисич'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '17dad93e-407f-5e1b-80a2-c84b13c0dba5', 'Галерея Anna Nova', 'ул. Жуковского, 28', '', 'Центральный', 59.936246, 30.359804
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Галерея Anna Nova'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '352fdf0d-9ab9-56e7-bccf-d90b7a96b7b7', 'Музей современного искусства «Артмуза»', '13-я линия В.О., 70', '', 'Василеостровский', 59.945686, 30.264865
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей современного искусства «Артмуза»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '113a361a-fa07-5197-932e-fabafe5e254b', 'Шанти Place', 'Мясницкий проезд, 2/1', 'Красные Ворота', 'Басманный', 55.768162, 37.64593
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Шанти Place'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0e0df21a-388a-5fac-afab-cd0b1f83cb8a', 'Zen Space (студия йоги, ЖК Саларьево Парк)', 'Саларьевская улица, 14, корп. 3', 'Саларьево', 'Новомосковский', 55.616702, 37.411059
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Zen Space (студия йоги, ЖК Саларьево Парк)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '318a9f00-60dc-5281-8f47-94284a010ab6', 'Вкус & Цвет (студия йоги и практик)', 'Большая Новодмитровская ул., 36, стр. 7', 'Дмитровская', 'Бутырский', 55.805864, 37.585034
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Вкус & Цвет (студия йоги и практик)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a1042eb8-a0bd-52f2-b6ef-a9ebb2df1e61', 'ДЫШИ студия йоги и цигуна', 'улица Фадеева, 4А', 'Маяковская / Новослободская', '', 55.775023, 37.600236
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ДЫШИ студия йоги и цигуна'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '213069ed-ae20-5dd4-8fb6-a0884fb7cdcb', 'UBU Yoga (студия горячей йоги)', 'Благовещенский переулок, 3, стр. 1', 'Маяковская', 'Пресненский (Патриаршие пруды)', 55.767734, 37.598064
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('UBU Yoga (студия горячей йоги)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '8d963d5a-a59d-56b2-85c7-5fc75c417d91', 'Планета Перемен (йога-центр на Мытнинской)', 'Мытнинская улица, 11', 'Площадь Восстания / Чернышевская', 'Центральный', 59.933512, 30.377733
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Планета Перемен (йога-центр на Мытнинской)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6fbcceb8-2f66-571b-94ee-687658fb7eea', 'Планета Перемен (йога-центр на Ушинского)', 'улица Ушинского, 3, корп. 3', '', 'Калининский', 60.040056, 30.414771
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Планета Перемен (йога-центр на Ушинского)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'bbb6030b-be52-5623-9a98-9dee0a3185d9', 'Praktika (студия йоги)', 'Большая Пушкарская улица, 10', 'Петроградская / Чкаловская', 'Петроградский', 59.955908, 30.298506
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Praktika (студия йоги)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '590c396c-83b6-58fc-9206-8494e86b79d1', 'Yoga Point (студия йоги на Петроградской)', 'Каменноостровский проспект, 10Б', 'Петроградская', 'Петроградский', 59.958701, 30.316095
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Yoga Point (студия йоги на Петроградской)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f81ab4fa-9fda-51c9-a956-015b41b9c9bb', 'Государственная Третьяковская галерея', 'Лаврушинский переулок, 10', 'Третьяковская', 'Замоскворечье', 55.741361, 37.62022
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственная Третьяковская галерея'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6b54a002-d19b-5b1e-84d8-2e06a1c4ac7b', 'Государственный исторический музей', 'Красная площадь, 1', 'Охотный Ряд / Площадь Революции', 'Тверской', 55.755323, 37.617882
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственный исторический музей'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '761ae84b-a1d7-56b3-8202-52b6b02c0208', 'Музей Москвы', 'Тверская улица, 21, стр. 1', 'Тверская / Пушкинская', 'Тверской', 55.73658, 37.593406
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей Москвы'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a10adf84-2728-5b90-903c-88a85fe8b8a0', 'Государственный биологический музей им. К.А. Тимирязева', 'Малая Грузинская улица, 15', 'Краснопресненская / Баррикадная', 'Пресненский', 55.764346, 37.57149
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственный биологический музей им. К.А. Тимирязева'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0f78f05a-7813-587d-ab42-19525970a8d1', 'Измайловский кремль (историко-культурный комплекс)', 'Измайловское шоссе, 73', 'Партизанская', 'Измайлово', 55.795463, 37.749761
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Измайловский кремль (историко-культурный комплекс)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '07c7115d-950c-5751-b197-61c03e387777', 'Государственный Эрмитаж', 'Дворцовая площадь, 2', 'Адмиралтейская', 'Центральный', 59.939043, 30.315354
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственный Эрмитаж'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '8dc34602-cbb4-5df9-ac21-b6c6f8ecf5ea', 'Государственный Русский музей', 'улица Инженерная, 4', 'Невский проспект / Гостиный двор', 'Центральный', 59.938711, 30.332311
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственный Русский музей'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '7fcd9c0b-1566-54c8-8cb2-6a43d25f61eb', 'Музей антропологии и этнографии им. Петра Великого (Кунсткамера)', 'Университетская набережная, 3', 'Адмиралтейская / Спортивная', 'Василеостровский', 59.942357, 30.304005
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Музей антропологии и этнографии им. Петра Великого (Кунсткамера)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'cc2f8d16-ba4d-5fe4-9557-97a984b3d5e9', 'Государственный музей истории Санкт-Петербурга (Петропавловская крепость)', 'Александровский парк, 7, Петропавловская крепость, 3И', 'Горьковская / Спортивная', 'Петроградский', 59.950375, 30.315291
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Государственный музей истории Санкт-Петербурга (Петропавловская крепость)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '476ba2af-364d-54a8-9339-51a422ba1af0', 'Большой театр', 'Театральная площадь, 1', 'Театральная / Охотный ряд', 'Тверской', 55.76013, 37.618612
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Большой театр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '0ff34d79-0969-5e84-8508-99a35509cec4', 'Малый театр', 'Театральный проезд, 1', 'Театральная / Охотный ряд', 'Тверской', 55.75975, 37.620631
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Малый театр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '81d2c0f7-8868-50df-83a9-457268d3bff6', 'Московский театр оперетты', 'улица Большая Дмитровка, 6/2', 'Театральная / Охотный ряд', 'Тверской', 55.764262, 37.611803
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский театр оперетты'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9e097736-496d-59a9-9cf7-34bbdf156eeb', 'Центральный театр кукол им. С.В. Образцова', 'Садовая-Самотечная улица, 3', 'Маяковская / Цветной бульвар', 'Тверской', 55.774039, 37.614306
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Центральный театр кукол им. С.В. Образцова'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '936e8f01-6dd9-5448-9ccb-b47e7d3e7775', 'Московский Художественный театр им. А.П. Чехова (МХТ)', 'Камергерский переулок, 3', 'Охотный ряд / Театральная', 'Тверской', 55.759913, 37.612753
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Московский Художественный театр им. А.П. Чехова (МХТ)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b08bfd51-a743-5d9d-b6dd-7a8fb813b4e5', 'Мариинский театр', 'Театральная площадь, 1', 'Садовая / Сенная площадь / Спасская', 'Адмиралтейский', 59.925393, 30.295836
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Мариинский театр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '23d67451-a225-5fd3-a6c7-31dd4063718e', 'Михайловский театр', 'площадь Искусств, 1', 'Невский проспект / Гостиный двор', 'Центральный', 59.937932, 30.329065
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Михайловский театр'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '1f525e1a-d3ce-5424-a137-6fee95fc1a99', 'Большой драматический театр им. Г.А. Товстоногова (БДТ)', 'Большая Конюшенная улица, 27', 'Невский проспект', 'Центральный', 59.937099, 30.321334
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Большой драматический театр им. Г.А. Товстоногова (БДТ)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c3047edf-3a56-5f5d-9c05-47b99f195ba4', 'Санкт-Петербургский государственный театр музыкальной комедии', 'Итальянская улица, 13', 'Невский проспект / Гостиный двор', 'Центральный', 59.936017, 30.332937
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Санкт-Петербургский государственный театр музыкальной комедии'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd8011821-d7c4-5157-b873-1d29c50ba9df', 'Парк искусств «Музеон»', 'Крымский Вал, владение 2', 'Октябрьская / Парк культуры', 'Якиманка', 55.735392, 37.607342
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Парк искусств «Музеон»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fc16252c-1bde-51e1-a125-07ca5e444d2b', 'Измайловский парк', 'аллея Большого Круга, 7/а', 'Партизанская / Измайлово (МЦК)', 'Измайлово', 55.772287, 37.754873
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Измайловский парк'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '65a97db9-d133-58d0-ac23-d4f492913d74', 'Парк «Сокольники»', '5-й Лучевой просек, район Сокольники', 'Сокольники', 'Сокольники', 55.816019, 37.675827
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Парк «Сокольники»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a66b24b6-77e1-58b9-b132-fd78081dfe47', 'Парк «Красная Пресня»', 'улица Мантулинская, 5', 'Выставочная / Улица 1905 года', 'Пресненский', 55.755914, 37.552797
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Парк «Красная Пресня»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '2b563440-ee97-5010-8c45-ca88108117cf', 'Новая Голландия (парк)', 'набережная Адмиралтейского канала, 2', 'Адмиралтейская', 'Центральный', 59.929392, 30.28741
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Новая Голландия (парк)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '5c20f752-e0b6-5284-b34c-c458fc3768e7', 'Елагин остров (ЦПКиО им. С.М. Кирова)', 'Елагин остров, 4', 'Крестовский остров / Старая Деревня', 'Петроградский', 59.979317, 30.270036
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Елагин остров (ЦПКиО им. С.М. Кирова)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6c2bf459-99ba-5728-8518-8e348ba48514', 'Парк Сосновка', 'улица Жака Дюкло, 20Б', 'Озерки / Удельная', 'Выборгский', 60.025758, 30.335167
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Парк Сосновка'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '53eb4fcb-7ff8-5895-943d-c018aec9117d', 'Циферблат (Ziferblat) на Кузнецком Мосту', 'ул. Кузнецкий Мост, 19, стр. 1', 'Кузнецкий Мост / Лубянка', '', 55.762614, 37.625437
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Циферблат (Ziferblat) на Кузнецком Мосту'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '03922497-e0dd-56b7-941d-98fbcf1a6368', 'Циферблат (Ziferblat) на Тверской', 'ул. Тверская, 12, стр. 1', 'Тверская / Пушкинская', '', 55.771925, 37.596468
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Циферблат (Ziferblat) на Тверской'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '97374623-3ef1-58b1-a61d-a83fa46e068a', 'Циферблат (Ziferblat) на Солянке', 'Лубянский проезд, 27/1, к1', '', '', 55.759296, 37.628192
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Циферблат (Ziferblat) на Солянке'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c8268a25-0657-53f7-a3a1-86a753bf6271', 'Книжный магазин-кафе «Достоевский»', 'Большой Трёхсвятительский переулок, 2/1, стр. 6', '', '', 55.754814, 37.645219
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Книжный магазин-кафе «Достоевский»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '162594ab-1baa-575e-b4f1-050a463d1c7d', 'Кафе Музея «Гараж»', 'Парк Горького, ул. Крымский Вал, 9, стр. 32', 'Октябрьская / Парк культуры', '', 55.731344, 37.603725
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Кафе Музея «Гараж»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6211248c-7581-5c0a-8a63-20e6282a6e55', 'Циферблат (Ziferblat) на Невском', 'Невский проспект, 81', 'Площадь Восстания', '', 59.930804, 30.359081
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Циферблат (Ziferblat) на Невском'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '768b9d2d-1000-5de1-9ffa-22ec05cd8791', 'CODE 505 (антикафе)', 'Невский проспект, 180 (5 этаж)', '', '', 59.925103, 30.382183
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('CODE 505 (антикафе)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '130b1d1b-3334-5d3e-a5af-0928d0ac57e8', 'Книжный магазин «Подписные издания»', 'Литейный проспект, 57', '', '', 59.934412, 30.347687
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Книжный магазин «Подписные издания»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'c63bd6e8-e43f-5d4c-98d9-1767513089a6', 'Книжный магазин «Порядок слов»', 'набережная реки Фонтанки, 15, 1 этаж', 'Гостиный двор', '', 59.936243, 30.34308
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Книжный магазин «Порядок слов»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'a2e22e39-c72e-5132-ba21-2c85e899756c', 'SOK Земляной Вал', 'Земляной Вал, 8', '', '', 55.762187, 37.656686
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('SOK Земляной Вал'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '02ca339e-a4f9-54e8-a535-ef0d009328b5', 'Коворкинг «Ключ» Патриаршие', 'Ермолаевский переулок (Патриаршие пруды)', 'Маяковская', '', 55.764484, 37.590911
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Коворкинг «Ключ» Патриаршие'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '1fe87216-e8da-54a5-b621-4dddde4c82a8', 'Коворкинг «Практик»', 'Воронцовская улица, 49/28, стр. 1', '', '', 55.732838, 37.666721
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Коворкинг «Практик»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '6a4efe99-9fa3-5176-b6ac-4e88637fafb3', 'Рабочая Станция (коворкинг)', 'Ленинский проспект, 30А', '', '', 55.711722, 37.581417
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Рабочая Станция (коворкинг)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'e2ace36f-2b24-5ef1-a5d2-7b1bd06d7486', 'Worknation (Москва-Сити)', 'Пресненская набережная, 12, башня «Федерация-Восток», 37 этаж', '', 'Москва-Сити', 55.749683, 37.537528
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Worknation (Москва-Сити)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fd39d9ba-9dda-5204-a45c-87ef4cd5c9f9', 'SMART-coworking на Коломяжском', 'Коломяжский проспект, 19, к2, ТК «Капитолий», 3 этаж', 'Пионерская', 'Приморский район', 60.007086, 30.297589
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('SMART-coworking на Коломяжском'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fd2926a4-10da-5d98-9da5-60254970038b', 'SMART-coworking на Шпалерной', 'Шпалерная улица, 60Б, 1 этаж', 'Чернышевская', 'Центральный район', 59.949026, 30.357777
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('SMART-coworking на Шпалерной'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '547e0252-24b8-5915-bd8f-5935e4678847', 'GrowUP Coworking', 'Большой Сампсониевский проспект, 61, к2', 'Выборгская', '', 59.973303, 30.342663
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('GrowUP Coworking'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'bef552e5-b43f-5dd6-9b2e-64074f76c70f', 'Культурный центр ЗИЛ', 'улица Восточная, 4, корпус 1', 'Автозаводская / Дубровка', '', 55.714469, 37.657881
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Культурный центр ЗИЛ'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'fb1d9821-4372-563d-8825-528d2588cd9a', 'ДК Рассвет', 'Столярный переулок, 3, к15', '', '', 55.764334, 37.567965
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ДК Рассвет'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '43cda7e0-f447-5194-b7e5-c17074aba9b1', 'Культурный центр «Москвич»', 'Волгоградский проспект, 46/15', '', '', 55.708076, 37.733283
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Культурный центр «Москвич»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '163b50e3-acef-5600-a942-0eec81c125b7', 'Центральный дом архитектора (ЦДА)', 'Гранатный переулок, 7, стр. 1', '', '', 55.759397, 37.590497
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Центральный дом архитектора (ЦДА)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b00b0313-fd24-5d36-84be-e1cfa49232cb', 'Культурный центр «ДОМ»', 'Большой Овчинниковский переулок, 24, строение 4', '', '', 55.744477, 37.651818
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Культурный центр «ДОМ»'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '22f3959f-aa9b-5801-9e0c-3a6775108006', 'Дворец культуры имени Ленсовета', 'Каменноостровский проспект, 42', 'Петроградская', '', 59.966977, 30.310077
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дворец культуры имени Ленсовета'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '81cd89a9-4195-5c54-843e-2b4343b7c4d5', 'ДК имени Кирова', 'Большой проспект В.О., 83', '', 'Васильевский остров', 59.933847, 30.255012
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('ДК имени Кирова'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'dd38abb5-b7e2-584b-9ead-814217bbd747', 'Дом учёных имени М. Горького', 'Дворцовая набережная, 26', '', '', 59.943151, 30.320119
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дом учёных имени М. Горького'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'b741635b-2fde-538b-bd8e-1c97d75b00c3', 'Дом культуры «Созвездие» (Калининский КДЦ)', 'Пискарёвский проспект, 10', '', 'Калининский район', 59.964895, 30.408383
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Дом культуры «Созвездие» (Калининский КДЦ)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '95a18f5a-525b-5fc8-b092-e8b0ec3ef73a', 'Концертный зал «У Финляндского» (Калининский КДЦ)', 'Арсенальная набережная, 13/1, литер А', '', 'Калининский район', 59.953266, 30.357512
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Концертный зал «У Финляндского» (Калининский КДЦ)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'ed16f7e5-e512-5f05-9295-4e6ed258ab1b', 'Лекторий Политехнического музея', '4-я Магистральная улица, 11, стр. 2 (БЦ «Магистраль»)', 'Полежаевская / Хорошёвская', '', 55.770445, 37.517008
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Лекторий Политехнического музея'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '9eb70c52-253d-521f-a5b6-027719fc3e18', 'НИУ ВШЭ — лекторий OpenLectures', 'Малый Гнездниковский переулок, 4, аудитория 10', '', '', 55.761819, 37.606169
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('НИУ ВШЭ — лекторий OpenLectures'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '123f5c07-7ef6-550c-afab-bb04d4cf177f', 'Точка кипения — Москва', 'Малый Конюшковский переулок, 2, 3 этаж', '', '', 55.75826, 37.580595
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Точка кипения — Москва'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'f129d60f-c656-5a46-ac9b-c703508984e8', 'Библиотека им. Н.А. Некрасова (конференц-зал/аудитории)', 'улица Бауманская, 58/25, строение 14', '', '', 55.767142, 37.678637
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Библиотека им. Н.А. Некрасова (конференц-зал/аудитории)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '5a1a6605-a48f-587c-97d8-1c3380a225cf', 'Точка кипения — Санкт-Петербург (ГУАП)', 'улица Большая Морская, 67', '', '', 59.930269, 30.295053
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Точка кипения — Санкт-Петербург (ГУАП)'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '705d9662-a02d-57e4-bf46-60ae3e153df5', 'Точка кипения МБИ', 'улица Малая Садовая, 6', '', '', 59.934784, 30.338025
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Точка кипения МБИ'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '2323fa5a-6696-5211-9b92-5c9477965740', 'Точка кипения Политех', 'Политехническая улица, 29, литер О', '', '', 60.009137, 30.381923
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Точка кипения Политех'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT 'd0931c3e-3a4a-5534-8b57-b524e7b046f1', 'Точка кипения РАНХиГС Санкт-Петербург', 'Каменноостровский проспект, 66 (корпус юридического факультета РАНХиГС)', '', '', 59.975484, 30.302215
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Точка кипения РАНХиГС Санкт-Петербург'));

INSERT INTO venues (id, name, address, metro, district, lat, lon)
SELECT '70efea3f-6fa3-5c9d-96e3-1099ff0a3c6c', 'Европейский университет в Санкт-Петербурге (публичные лекции)', 'Гагаринская улица, 3/2 (главное здание, Шпалерная ул., 1)', '', '', 59.944614, 30.342909
WHERE NOT EXISTS (SELECT 1 FROM venues WHERE lower(name) = lower('Европейский университет в Санкт-Петербурге (публичные лекции)'));

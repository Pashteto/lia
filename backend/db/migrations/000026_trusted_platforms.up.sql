-- 000026_trusted_platforms.up.sql
-- Trusted external-registration platforms (spec 2026-08-12). domain_suffix is
-- stored in punycode, no scheme; matching is exact host or dot-boundary suffix.
CREATE TABLE IF NOT EXISTS trusted_platforms (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_suffix text NOT NULL UNIQUE,
    display_name  text NOT NULL,
    category      text NOT NULL CHECK (category IN ('ticketing', 'afisha', 'gov', 'social')),
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE events ADD COLUMN IF NOT EXISTS capacity_limited boolean NOT NULL DEFAULT false;
ALTER TABLE events ADD COLUMN IF NOT EXISTS external_url_verified boolean NOT NULL DEFAULT false;

INSERT INTO trusted_platforms (domain_suffix, display_name, category) VALUES
  ('timepad.ru',            'TimePad',          'ticketing'),
  ('afisha.yandex.ru',      'Яндекс Афиша',     'afisha'),
  ('kassir.ru',             'Кассир.ру',        'ticketing'),
  ('qtickets.ru',           'Qtickets',         'ticketing'),
  ('qtickets.events',       'Qtickets',         'ticketing'),
  ('ticketscloud.com',      'Ticketscloud',     'ticketing'),
  ('ticketscloud.org',      'Ticketscloud',     'ticketing'),
  ('intickets.ru',          'Intickets',        'ticketing'),
  ('radario.ru',            'Радарио',          'ticketing'),
  ('ticketland.ru',         'Ticketland',       'ticketing'),
  ('ponominalu.ru',         'Ponominalu',       'ticketing'),
  ('live.mts.ru',           'МТС Live',         'ticketing'),
  ('afisha.ru',             'Афиша',            'afisha'),
  ('kinopoisk.ru',          'Кинопоиск',        'afisha'),
  ('vk.com',                'ВКонтакте',        'social'),
  ('events.nethouse.ru',    'Nethouse.События', 'ticketing'),
  ('leader-id.ru',          'Leader-ID',        'gov'),
  ('culture.ru',            'Культура.РФ',      'gov'),
  ('xn--80atdujec4e.xn--p1ai',   'Культура.РФ',      'gov'),
  ('mos.ru',                'mos.ru',           'gov'),
  ('vmuzey.com',            'ВМузей',           'afisha'),
  ('kudago.com',            'KudaGo',           'afisha')
ON CONFLICT (domain_suffix) DO NOTHING;

CREATE TABLE complaints (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type      TEXT NOT NULL DEFAULT 'event'
                     CHECK (target_type IN ('event')),
    target_id        uuid NOT NULL,
    reporter_user_id uuid NOT NULL,
    category         TEXT NOT NULL
                     CHECK (category IN ('spam','fraud','inappropriate','duplicate','other')),
    note             TEXT,
    status           TEXT NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open','resolved','dismissed')),
    resolution       TEXT,
    resolved_by      uuid,
    resolved_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- One OPEN complaint per reporter per target (repeat-submit dedup).
CREATE UNIQUE INDEX complaints_one_open_per_reporter
    ON complaints (target_type, target_id, reporter_user_id) WHERE status = 'open';

-- Fast "events with open complaints" grouping.
CREATE INDEX complaints_open_target_idx
    ON complaints (target_type, target_id) WHERE status = 'open';

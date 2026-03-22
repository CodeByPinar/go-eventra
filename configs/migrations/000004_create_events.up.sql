CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    organizer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_date TIMESTAMPTZ NOT NULL,
    location TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    participant_limit INT,
    category TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT participant_limit_non_negative CHECK (participant_limit IS NULL OR participant_limit >= 0)
);

CREATE INDEX IF NOT EXISTS idx_events_event_date ON events(event_date);
CREATE INDEX IF NOT EXISTS idx_events_visibility_event_date ON events(visibility, event_date DESC);
CREATE INDEX IF NOT EXISTS idx_events_organizer_id ON events(organizer_id);
CREATE INDEX IF NOT EXISTS idx_events_category ON events(category);
CREATE INDEX IF NOT EXISTS idx_events_location ON events(location);
CREATE INDEX IF NOT EXISTS idx_events_tags_gin ON events USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_events_title_trgm ON events USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_events_description_trgm ON events USING GIN (description gin_trgm_ops);

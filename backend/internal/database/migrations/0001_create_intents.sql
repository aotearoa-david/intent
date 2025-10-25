CREATE TABLE IF NOT EXISTS intents (
    id UUID PRIMARY KEY,
    statement TEXT NOT NULL,
    context TEXT NOT NULL,
    expected_outcome TEXT NOT NULL,
    collaborators JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

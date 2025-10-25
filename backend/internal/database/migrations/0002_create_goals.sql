CREATE TABLE IF NOT EXISTS goals (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    clarity_statement TEXT NOT NULL,
    constraints JSONB NOT NULL DEFAULT '[]'::jsonb,
    success_criteria JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

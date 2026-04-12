CREATE TABLE IF NOT EXISTS links
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_url VARCHAR(255) NOT NULL,
    original_url VARCHAR(255) NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_short_url ON links(short_url);

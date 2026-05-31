ALTER TABLE links add column user_id VARCHAR(255);
CREATE INDEX idx_links_user_id ON links(user_id);
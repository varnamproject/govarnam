CREATE TABLE IF NOT EXISTS metadata (
  key TEXT UNIQUE,
  value TEXT
);

CREATE TABLE IF NOT EXISTS words (
  id INTEGER PRIMARY KEY,
  word TEXT UNIQUE,
  weight INTEGER DEFAULT 1,
  learned_on INTEGER
);

CREATE TABLE IF NOT EXISTS patterns (
  pattern TEXT NOT NULL COLLATE NOCASE,
  word_id INTEGER NOT NULL,
  FOREIGN KEY(word_id) REFERENCES words(id) ON DELETE CASCADE,
  PRIMARY KEY(pattern, word_id)
);

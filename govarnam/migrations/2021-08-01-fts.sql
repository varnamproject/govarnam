-- Note: FTS can't be applied on patterns because
-- we require partial word search which FTS doesn't support

CREATE VIRTUAL TABLE IF NOT EXISTS words_fts USING FTS5(
  word,
  weight UNINDEXED,
  learned_on UNINDEXED,
  content='words',
  content_rowid='id',
  tokenize='ascii',
  prefix='1 2',
);
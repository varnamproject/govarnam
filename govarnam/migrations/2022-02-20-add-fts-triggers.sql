-- FTS Triggers was added on August 2021 but the delete triggers wasn't being executed
-- https://github.com/varnamproject/govarnam/issues/24

CREATE TRIGGER IF NOT EXISTS words_ai AFTER INSERT ON words
  BEGIN
    INSERT INTO words_fts (rowid, word)
    VALUES (new.id, new.word);
  END;

CREATE TRIGGER IF NOT EXISTS words_ad AFTER DELETE ON words
  BEGIN
    INSERT INTO words_fts (words_fts, rowid, word)
    VALUES ('delete', old.id, old.word);
  END;

CREATE TRIGGER IF NOT EXISTS words_au AFTER UPDATE ON words
  BEGIN
    INSERT INTO words_fts (words_fts, rowid, word)
    VALUES ('delete', old.id, old.word);
    INSERT INTO words_fts (rowid, word)
    VALUES (new.id, new.word);
  END;
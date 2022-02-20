package govarnam

import (
	"database/sql"
	"embed"
	"io/fs"
	"testing"
)

//go:embed testdata/*.sql
var testdataFS embed.FS

func TestMigration(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")

	checkError(err)

	testdataDirFS, err := fs.Sub(testdataFS, "testdata")
	checkError(err)

	dirFiles, err := fs.ReadDir(testdataDirFS, ".")
	checkError(err)

	mg, err := InitMigrate(db, testdataDirFS)
	checkError(err)

	_, err = db.Query("SELECT * FROM words")
	assertEqual(t, err != nil, true)

	ranMigrations, err := mg.Run()
	assertEqual(t, err, nil)
	assertEqual(t, ranMigrations, len(dirFiles))

	_, err = db.Query("SELECT * FROM words")
	assertEqual(t, err, nil)

	// Part 2 : New Migrations

	_, err = db.Query("SELECT * FROM words_fts")
	assertEqual(t, err != nil, true)

	migrationsFS, err := fs.Sub(embedFS, "migrations")
	checkError(err)

	dirFiles, err = fs.ReadDir(migrationsFS, ".")
	checkError(err)

	mg, err = InitMigrate(db, migrationsFS)
	checkError(err)

	ranMigrations, err = mg.Run()
	assertEqual(t, err, nil)
	assertEqual(t, ranMigrations, len(dirFiles))

	_, err = db.Query("SELECT * FROM words_fts")
	assertEqual(t, err, nil)
}

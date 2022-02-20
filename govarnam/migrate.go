package govarnam

import (
	sql "database/sql"
	"io/fs"
	"strings"
)

type migrate struct {
	db *sql.DB
	fs fs.FS
}

type migrationStatus struct {
	lastRun       string
	lastMigration string
}

func InitMigrate(db *sql.DB, fs fs.FS) (*migrate, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			name VARCHAR(200)
		);
	`)
	if err != nil {
		return nil, err
	}

	return &migrate{db, fs}, nil
}

func (mg *migrate) Status() (*migrationStatus, error) {
	var lastRun string = ""
	mg.db.QueryRow("SELECT name FROM migrations ORDER BY id DESC LIMIT 1").Scan(&lastRun)

	files, err := fs.ReadDir(mg.fs, ".")
	if err != nil {
		return nil, err
	}

	lastMigration := files[len(files)-1].Name()

	return &migrationStatus{lastRun, lastMigration}, nil
}

func (mg *migrate) Run() (int, error) {
	ranMigrations := 0

	status, err := mg.Status()
	if err != nil {
		return 0, err
	}

	if status.lastRun != status.lastMigration {
		ranMigrations, err = mg.runMigrations(status)
		if err != nil {
			return 0, err
		}
	}

	return ranMigrations, nil
}

func (mg *migrate) runMigrations(status *migrationStatus) (int, error) {
	files, err := fs.ReadDir(mg.fs, ".")
	if err != nil {
		return 0, err
	}

	ranMigrations := 0

	// lastRun will be empty if no migrations have been run
	var foundLastRunMigration bool = (status.lastRun == "")

	for _, file := range files {
		fileNameParts := strings.Split(file.Name(), ".")
		migrationName := fileNameParts[0]

		// Run all migrations after the last ran migration
		if !foundLastRunMigration {
			foundLastRunMigration = (status.lastRun == migrationName)
		} else {
			fileContents, err := fs.ReadFile(mg.fs, file.Name())
			if err != nil {
				return 0, err
			}

			tx, err := mg.db.Begin()
			if err != nil {
				return 0, err
			}

			_, err = tx.Exec(string(fileContents))
			if err != nil {
				return 0, err
			}

			stmt, err := tx.Prepare("INSERT INTO migrations (name) VALUES(?)")
			if err != nil {
				tx.Rollback()
				return 0, err
			}
			_, err = stmt.Exec(migrationName)
			if err != nil {
				tx.Rollback()
			}

			tx.Commit()

			ranMigrations++
		}
	}

	return ranMigrations, nil
}

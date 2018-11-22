package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/mysql"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/golang-migrate/migrate/source/github"
	log "github.com/sirupsen/logrus"
	"os"
)

var dbConnString = os.Getenv("DB_CONNECTION_STRING")
var migrationVer uint = 1
var db *sql.DB
var hystrixDb = "db"
var campStmt *sql.Stmt
var addCampStmt *sql.Stmt

func newMigrate() (*migrate.Migrate, error) {
	return migrate.New(
		getEnv("MIGRATIONS_SOURCE", "file://migrations"),
		"mysql://"+dbConnString)
}

func migrateDatabase() {
	log.Info("migrating database")
	m, err := newMigrate()
	if err != nil {
		log.Fatal("failed to connect ", err)
	}
	defer m.Close()

	err = m.Migrate(migrationVer)
	if err != nil && err.Error() != "no change" {
		log.Fatal("failed to migrate ", err)
	}
}

func dropDatabase() {
	log.Info("dropping database")
	m, err := newMigrate()
	if err != nil {
		log.Fatal("failed to connect ", err)
	}
	defer m.Close()

	err = m.Drop()
	if err != nil && err.Error() != "no change" {
		log.Fatal("failed to drop ", err)
	}
}

func initializeDatabase() {
	var err error
	db, err = sql.Open("mysql", dbConnString+"?charset=utf8mb4,utf8")
	if err != nil {
		log.Fatal("failed to open sql ", err)
	}

	campStmt, err = db.Prepare(
		"select `id`, `title`, `url`, `image`, `ratio`, `placeholder`, `source`" +
			"from `ads` where `start` <= ? and end > ?")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}

	addCampStmt, err = db.Prepare(
		"insert into `ads` " +
			"(`id`, `title`, `url`, `image`, `ratio`, `placeholder`, `source`, `start`, `end`) " +
			"values (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}
}

func tearDatabase() {
	addCampStmt.Close()
	campStmt.Close()

	db.Close()
}

package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"
	_ "github.com/golang-migrate/migrate/database/mysql"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/golang-migrate/migrate/source/github"
	log "github.com/sirupsen/logrus"
	"os"
)

var dbConnString = os.Getenv("DB_CONNECTION_STRING")
var migrationVer uint = 3
var db *sql.DB
var hystrixDb = "db"
var campStmt *sql.Stmt
var addCampStmt *sql.Stmt

func openDatabaseConnection() (*sql.DB, error) {
	return sql.Open("mysql", dbConnString+"?charset=utf8mb4,utf8")
}

func newMigrate() (*migrate.Migrate, error) {
	con, err := openDatabaseConnection()
	if err != nil {
		log.Fatal("failed to open sql ", err)
	}
	driver, _ := mysql.WithInstance(con, &mysql.Config{})
	return migrate.NewWithDatabaseInstance(
		getEnv("MIGRATIONS_SOURCE", "file://migrations"),
		"mysql", driver)
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
	db, err = openDatabaseConnection()
	if err != nil {
		log.Fatal("failed to open sql ", err)
	}

	campStmt, err = db.Prepare(
		"select `id`, `title`, `url`, `image`, `ratio`, `placeholder`, " +
			"`source`, `company`, `probability`, `fallback`, `geo`" +
			"from `ads` where `start` <= ? and end > ?")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}

	addCampStmt, err = db.Prepare(
		"insert into `ads` " +
			"(`id`, `title`, `url`, `image`, `ratio`, `placeholder`, `source`, " +
			"`company`, `probability`, `fallback`, `geo`, `start`, `end`) " +
			"values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}
}

func tearDatabase() {
	addCampStmt.Close()
	campStmt.Close()

	db.Close()
}

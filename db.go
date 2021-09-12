package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	log "github.com/sirupsen/logrus"
	"os"
)

var dbConnString = os.Getenv("DB_CONNECTION_STRING")

const migrationVer uint = 7

var db *sql.DB
var hystrixDb = "db"
var campStmt *sql.Stmt
var addCampStmt *sql.Stmt
var getUserTagsStmt *sql.Stmt

func openDatabaseConnection() (*sql.DB, error) {
	return sql.Open("mysql", dbConnString+"?charset=utf8mb4,utf8")
}

func newMigrate() (*migrate.Migrate, error) {
	con, err := openDatabaseConnection()
	if err != nil {
		log.Fatal("failed to open sql ", err)
	}
	driver, err := mysql.WithInstance(con, &mysql.Config{})
	if err != nil {
		log.Fatal("failed to get driver ", err)
	}
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

	campStmt, err = db.Prepare("select id, title, url, image, ratio, placeholder, source, company, probability, fallback, geo from ads left join (select ad_id, max(relevant) as relevant from (select ad_id, exists (select user_id from user_tags where user_tags.tag = ad_tags.tag and user_tags.user_id = ?) as relevant from ad_tags) as res group by ad_id) relevant_ads on ads.id = relevant_ads.ad_id where `start` <= ? and end > ? and (relevant_ads.relevant = 1 or relevant_ads.ad_id is null)")
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

	getUserTagsStmt, err = db.Prepare("select tag from user_tags where user_id = ?")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}
}

func tearDatabase() {
	addCampStmt.Close()
	campStmt.Close()
	getUserTagsStmt.Close()

	db.Close()
}

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
	"time"
)

var dbConnString = os.Getenv("DB_CONNECTION_STRING")

const migrationVer uint = 11

var db *sql.DB
var hystrixDb = "db"
var campStmt *sql.Stmt
var addCampStmt *sql.Stmt
var getUserTagsStmt *sql.Stmt
var getUserExperienceLevelStmt *sql.Stmt

func openDatabaseConnection() (*sql.DB, error) {
	conn, err := sql.Open("mysql", dbConnString+"?charset=utf8mb4,utf8")
	if err != nil {
		return nil, err
	}

	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(20)

	return conn, nil
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

	campStmt, err = db.Prepare(`
		select id,
		   title,
		   url,
		   image,
		   ratio,
		   placeholder,
		   source,
		   company,
		   probability,
		   fallback,
		   geo,
		   tag_relevant_ads.ad_id is not null as is_tag_targeted,
		   exp_relevant_ads.ad_id is not null as is_exp_targeted
		from ads
         	left join (select ad_id, max(relevant) as relevant  
                    from (select ad_id,
                                 exists (select user_id
                                         from user_tags
                                         where user_tags.tag = ad_tags.tag
                                           and user_tags.user_id = ?) as relevant
                          from ad_tags) as res
                    group by ad_id) tag_relevant_ads on ads.id = tag_relevant_ads.ad_id
         	left join (select ad_id, max(relevant) as relevant
                    from (select ad_id,
                                 exists (select user_id
                                         from user_experience_levels
                                         where user_experience_levels.experience_level = ad_experience_level.experience_level
                                           and user_experience_levels.user_id = ?) as relevant
                          from ad_experience_level) as res
                    group by ad_id) exp_relevant_ads on ads.id = exp_relevant_ads.ad_id
		where start <= ? and end > ? and 
		      (
      			(tag_relevant_ads.relevant = 1 and exp_relevant_ads.relevant = 1)
    			or
       			(tag_relevant_ads.relevant is null and exp_relevant_ads.relevant is null)
    			or
       			(tag_relevant_ads.relevant = 1 and exp_relevant_ads.relevant is null)
    			or
       			(tag_relevant_ads.relevant is null and exp_relevant_ads.relevant = 1)
    		  )`)
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

	getUserTagsStmt, err = db.Prepare("select tag from user_tags where user_id = ? order by last_read desc limit 50")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}

	getUserExperienceLevelStmt, err = db.Prepare("select experience_level from user_experience_levels where user_id = ?")
	if err != nil {
		log.Fatal("failed to prepare query ", err)
	}
}

func tearDatabase() {
	addCampStmt.Close()
	campStmt.Close()
	getUserTagsStmt.Close()
	getUserExperienceLevelStmt.Close()

	db.Close()
}

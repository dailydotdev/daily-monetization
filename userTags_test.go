package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddOrUpdateUserTags(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()
	_, err := db.Exec("INSERT INTO user_tags (user_id, tag, last_read) VALUES ('1', 'webdev', '2021-09-12 08:54:07')")
	assert.Nil(t, err)

	rows, err := db.Query("SELECT last_read FROM user_tags WHERE tag = 'webdev' LIMIT 1")
	assert.Nil(t, err)
	defer rows.Close()
	rows.Next()
	var webdevLastRead string
	err = rows.Scan(&webdevLastRead)
	assert.Nil(t, err)

	err = addOrUpdateUserTags(context.Background(), "1", []string{"webdev", "javascript"})
	assert.Nil(t, err)

	rows, err = db.Query("SELECT user_id, tag, last_read FROM user_tags ORDER BY tag")
	assert.Nil(t, err)
	defer rows.Close()

	var userId string
	var tag string
	var lastRead string
	var i = 0
	for rows.Next() {
		err := rows.Scan(&userId, &tag, &lastRead)
		assert.Nil(t, err)
		assert.Equal(t, "1", userId)
		if i == 0 {
			assert.Equal(t, "javascript", tag)
		} else if i == 1 {
			assert.Equal(t, "webdev", tag)
			assert.NotEqual(t, lastRead, webdevLastRead)
		}
		i++
	}
	assert.Equal(t, 2, i)
	err = rows.Err()
	assert.Nil(t, err)
}

func TestDeleteOldUserTags(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()
	_, err := db.Exec("INSERT INTO user_tags (user_id, tag, last_read) VALUES ('1', 'webdev', '2021-01-12 08:54:07')")
	assert.Nil(t, err)

	err = addOrUpdateUserTags(context.Background(), "1", []string{"php", "javascript"})
	assert.Nil(t, err)

	err = deleteOldTags(context.Background())
	assert.Nil(t, err)

	rows, err := db.Query("SELECT count(*) FROM user_tags")
	assert.Nil(t, err)
	defer rows.Close()
	rows.Next()
	var count int
	rows.Scan(&count)
	assert.Equal(t, 2, count)
}
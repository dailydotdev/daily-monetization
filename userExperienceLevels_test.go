package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetOrUpdateUserExperienceLevel(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()
	_, err := db.Exec("INSERT INTO user_experience_levels (user_id, experience_level) VALUES ('1', 'MORE_THAN_2_YEARS')")
	require.NoError(t, err)

	err = setOrUpdateExperienceLevel(context.Background(), "1", "MORE_THAN_6_YEARS")
	require.NoError(t, err)

	row := db.QueryRow("SELECT user_id, experience_level FROM user_experience_levels")
	require.NoError(t, err)
	var userId string
	var experienceLevel string

	err = row.Scan(&userId, &experienceLevel)
	require.NoError(t, err)
	require.Equal(t, "MORE_THAN_6_YEARS", experienceLevel)
}

func TestDeleteUserExperienceLevel(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()
	_, err := db.Exec("INSERT INTO user_experience_levels (user_id, experience_level) VALUES ('1', 'MORE_THAN_4_YEARS'), ('2', 'MORE_THAN_10_YEARS')")
	require.NoError(t, err)

	err = setOrUpdateExperienceLevel(context.Background(), "1", "MORE_THAN_10_YEARS")
	require.NoError(t, err)

	err = deleteUserExperienceLevel(context.Background(), "1")
	require.NoError(t, err)

	rows, err := db.Query("SELECT count(*) FROM user_experience_levels")
	require.NoError(t, err)
	defer rows.Close()
	rows.Next()
	var count int
	require.NoError(t, rows.Scan(&count))
	require.Equal(t, 1, count)
}

func TestGetUserExperienceLevel(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()
	_, err := db.Exec("INSERT INTO user_experience_levels (user_id, experience_level) VALUES ('1', 'MORE_THAN_4_YEARS'), ('2', 'MORE_THAN_10_YEARS')")
	require.NoError(t, err)

	{
		level, err := getUserExperienceLevel(context.Background(), "1")
		require.NoError(t, err)
		require.Equal(t, "MORE_THAN_4_YEARS", level)
	}

	{
		level, err := getUserExperienceLevel(context.Background(), "2")
		require.NoError(t, err)
		require.Equal(t, "MORE_THAN_10_YEARS", level)
	}

	{
		level, err := getUserExperienceLevel(context.Background(), "3")
		require.NoError(t, err)
		require.Equal(t, "UNKNOWN", level)
	}
}

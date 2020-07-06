package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptySegment(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()

	var res string
	var err error
	res, err = findSegment(context.Background(), "1")
	assert.Nil(t, err)
	assert.Equal(t, res, "")
}

func TestAddAndFetchSegment(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()

	err := updateUserSegment(context.Background(), "1", "frontend")
	assert.Nil(t, err)

	var res string
	res, err = findSegment(context.Background(), "1")
	assert.Nil(t, err)
	assert.Equal(t, res, "frontend")
}

func TestUpdateSegment(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()

	err := updateUserSegment(context.Background(), "1", "frontend")
	assert.Nil(t, err)

    err = updateUserSegment(context.Background(), "1", "backend")
    assert.Nil(t, err)

	var res string
	res, err = findSegment(context.Background(), "1")
	assert.Nil(t, err)
	assert.Equal(t, res, "backend")
}
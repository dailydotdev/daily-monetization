package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/afex/hystrix-go/hystrix"
)

func setOrUpdateExperienceLevel(ctx context.Context, userId string, experienceLevel string) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			var query = "INSERT INTO user_experience_levels (user_id, experience_level) VALUES (?, ?) ON DUPLICATE KEY UPDATE experience_level=?, d_update=CURRENT_TIMESTAMP"
			_, err := db.ExecContext(ctx, query, userId, experienceLevel, experienceLevel)
			if err != nil {
				return err
			}
			return nil
		}, nil)
}

func deleteUserExperienceLevel(ctx context.Context, userId string) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			_, err := db.ExecContext(ctx, "DELETE FROM user_experience_levels WHERE user_id = ?", userId)
			if err != nil {
				return err
			}
			return nil
		}, nil)
}

var getUserExperienceLevel = func(ctx context.Context, userId string) (string, error) {
	output := make(chan string, 1)
	errors := hystrix.GoC(ctx, hystrixDb,
		func(ctx context.Context) error {

			var experienceLevel string
			row := getUserExperienceLevelStmt.QueryRow(userId)
			switch err := row.Scan(&experienceLevel); {
			case errors.Is(err, sql.ErrNoRows):
				output <- "UNKNOWN"
				return nil
			case err == nil:
				output <- experienceLevel
				return nil
			default:
				return err
			}
		}, nil)
	select {
	case out := <-output:
		return out, nil
	case err := <-errors:
		return "UNKNOWN", err
	}
}

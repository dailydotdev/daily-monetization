package main

import (
	"context"
	"github.com/afex/hystrix-go/hystrix"
)

func addOrUpdateUserTags(ctx context.Context, userId string, tags []string) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			var parameters []interface{}
			var query = "INSERT INTO user_tags (user_id, tag) VALUES "
			for i, tag := range tags {
				if i > 0 {
					query += ", "
				}
				query += "(?,?)"
				parameters = append(parameters, userId, tag)
			}
			query += "ON DUPLICATE KEY UPDATE last_read=CURRENT_TIMESTAMP"
			_, err := db.ExecContext(ctx, query, parameters...)
			if err != nil {
				return err
			}
			return nil
		}, nil)
}

func deleteOldTags(ctx context.Context) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			_, err := db.ExecContext(ctx, "DELETE FROM user_tags WHERE last_read < now() - interval 6 month")
			if err != nil {
				return err
			}
			return nil
		}, nil)
}

var getUserTags = func(ctx context.Context, userId string) ([]string, error) {
	output := make(chan []string, 1)
	errors := hystrix.GoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			rows, err := getUserTagsStmt.QueryContext(ctx, userId)
			if err != nil {
				return err
			}
			defer rows.Close()

			var res []string
			var tag string
			for rows.Next() {
				err = rows.Scan(&tag)
				if err != nil {
					return err
				}
				res = append(res, tag)
			}
			err = rows.Err()
			if err != nil {
				return err
			}

			output <- res
			return nil
		}, nil)
	select {
	case out := <-output:
		return out, nil
	case err := <-errors:
		return nil, err
	}
}

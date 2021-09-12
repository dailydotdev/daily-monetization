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

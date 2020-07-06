package main

import (
	"context"
	"github.com/afex/hystrix-go/hystrix"
)

var updateUserSegment = func(ctx context.Context, userId string, segment string) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			_, err := updateSegmentStmt.ExecContext(ctx, userId, segment, segment)
			if err != nil {
				return err
			}
			return nil
		}, nil)
}

var findSegment = func(ctx context.Context, userId string) (string, error) {
	output := make(chan string, 1)
	errors := hystrix.GoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			rows, err := findSegmentStmt.QueryContext(ctx, userId)
			if err != nil {
				return err
			}
			defer rows.Close()

			var segment string = ""
			for rows.Next() {
				err := rows.Scan(&segment)
				if err != nil {
					return err
				}
			}
			err = rows.Err()
			if err != nil {
				return err
			}

			output <- segment
			return nil
		}, nil)

	select {
	case out := <-output:
		return out, nil
	case err := <-errors:
		return "", err
	}
}

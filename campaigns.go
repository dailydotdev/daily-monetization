package main

import (
	"context"
	"github.com/afex/hystrix-go/hystrix"
	"time"
)

type CampaignAd struct {
	Ad
	Id          string
	Placeholder string
	Ratio       float32
}

var addCampaign = func(ctx context.Context, camp CampaignAd, start time.Time, end time.Time) error {
	return hystrix.Do(hystrixDb,
		func() error {
			_, err := addCampStmt.ExecContext(ctx, camp.Id, camp.Description, camp.Link, camp.Image, camp.Ratio, camp.Placeholder, camp.Source, start, end)
			if err != nil {
				return err
			}

			return nil
		}, nil)
}

var fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
	output := make(chan []CampaignAd, 1)
	errors := hystrix.Go(hystrixDb,
		func() error {
			rows, err := campStmt.QueryContext(ctx, timestamp, timestamp)
			if err != nil {
				return err
			}
			defer rows.Close()

			var res []CampaignAd
			for rows.Next() {
				var camp CampaignAd
				err := rows.Scan(&camp.Id, &camp.Description, &camp.Link, &camp.Image, &camp.Ratio, &camp.Placeholder, &camp.Source)
				if err != nil {
					return err
				}
				res = append(res, camp)
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

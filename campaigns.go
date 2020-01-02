package main

import (
	"context"
	"database/sql"
	"github.com/afex/hystrix-go/hystrix"
	"time"
)

type Ad struct {
	Description string
	Image       string
	Link        string
	Source      string
	Company     string
}

type CampaignAd struct {
	Ad
	Id          string
	Placeholder string
	Ratio       float32
	Probability float32
	Fallback    bool
	Geo         string
}

type ScheduledCampaignAd struct {
	CampaignAd
	Start time.Time
	End   time.Time
}

var addCampaign = func(ctx context.Context, camp ScheduledCampaignAd) error {
	return hystrix.DoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			_, err := addCampStmt.ExecContext(ctx, camp.Id, camp.Description, camp.Link, camp.Image, camp.Ratio, camp.Placeholder, camp.Source, camp.Company, camp.Probability, camp.Fallback, camp.Geo, camp.Start, camp.End)
			if err != nil {
				return err
			}

			return nil
		}, nil)
}

var fetchCampaigns = func(ctx context.Context, timestamp time.Time) ([]CampaignAd, error) {
	output := make(chan []CampaignAd, 1)
	errors := hystrix.GoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			rows, err := campStmt.QueryContext(ctx, timestamp, timestamp)
			if err != nil {
				return err
			}
			defer rows.Close()

			var res []CampaignAd
			for rows.Next() {
				var camp CampaignAd
				var geo sql.NullString
				err := rows.Scan(&camp.Id, &camp.Description, &camp.Link, &camp.Image, &camp.Ratio, &camp.Placeholder, &camp.Source, &camp.Company, &camp.Probability, &camp.Fallback, &geo)
				if err != nil {
					return err
				}
				if geo.Valid {
					camp.Geo = geo.String
				} else {
					camp.Geo = ""
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

package main

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

type Ad struct {
	Description string
	Image       string
	Link        string
	Source      string
	Company     string
	ProviderId  string
}

type CampaignAd struct {
	Ad
	Id            string
	Placeholder   string
	Ratio         float32
	Probability   float32 `json:"-"`
	Fallback      bool    `json:"-"`
	Geo           string  `json:"-"`
	IsTagTargeted bool    `json:"-"`
	IsExpTargeted bool    `json:"-"`
}

type ScheduledCampaignAd struct {
	CampaignAd
	Start time.Time
	End   time.Time
}

var cloudinaryRegex = regexp.MustCompile(`(?:res\.cloudinary\.com\/daily-now|daily-now-res\.cloudinary\.com)`)

func mapCloudinaryUrl(url string) string {
	return cloudinaryRegex.ReplaceAllString(url, "media.daily.dev")
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

var fetchCampaigns = func(ctx context.Context, timestamp time.Time, userId string) ([]CampaignAd, error) {
	output := make(chan []CampaignAd, 1)
	errors := hystrix.GoC(ctx, hystrixDb,
		func(ctx context.Context) error {
			rows, err := campStmt.QueryContext(ctx, userId, userId, timestamp, timestamp)
			if err != nil {
				return err
			}
			defer rows.Close()

			var res []CampaignAd
			for rows.Next() {
				var camp CampaignAd
				var geo sql.NullString
				err = rows.Scan(&camp.Id, &camp.Description, &camp.Link, &camp.Image, &camp.Ratio, &camp.Placeholder, &camp.Source, &camp.Company, &camp.Probability, &camp.Fallback, &geo, &camp.IsTagTargeted, &camp.IsExpTargeted)
				if err != nil {
					return err
				}
				camp.Image = mapCloudinaryUrl(camp.Image)
				if geo.Valid && len(geo.String) > 0 {
					camp.Geo = geo.String
					if !camp.Fallback {
						if camp.IsTagTargeted || camp.IsExpTargeted {
							camp.ProviderId = "direct-combined"
						} else {
							camp.ProviderId = "direct-geo"
						}
					}
				} else {
					camp.Geo = ""
					if !camp.Fallback {
						if camp.IsTagTargeted || camp.IsExpTargeted {
							camp.ProviderId = "direct-keywords"
						} else {
							camp.ProviderId = "direct"
						}
					}
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

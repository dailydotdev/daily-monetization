package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var camp = CampaignAd{
	Placeholder: "placholder",
	Ratio:       0.5,
	Id:          "id",
	Probability: 1,
	Fallback:    true,
	Ad: Ad{
		Source:      "source",
		Image:       "image",
		Link:        "http://link.com",
		Description: "desc",
		Company:     "company",
	},
}

func TestAddAndFetchCampaigns(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()

	err := addCampaign(context.Background(), ScheduledCampaignAd{
		CampaignAd: camp,
		Start:      time.Now().Add(time.Hour * -1),
		End:        time.Now().Add(time.Hour),
	})
	assert.Nil(t, err)

	var res []CampaignAd
	res, err = fetchCampaigns(context.Background(), time.Now())
	assert.Nil(t, err)
	assert.Equal(t, []CampaignAd{camp}, res)
}

func TestFetchExpiredCampaigns(t *testing.T) {
	migrateDatabase()
	initializeDatabase()
	defer tearDatabase()
	defer dropDatabase()

	err := addCampaign(context.Background(), ScheduledCampaignAd{
		CampaignAd: camp,
		Start:      time.Now().Add(time.Hour * -2),
		End:        time.Now().Add(time.Hour * -1),
	})
	assert.Nil(t, err)

	var res []CampaignAd
	res, err = fetchCampaigns(context.Background(), time.Now())
	assert.Nil(t, err)
	assert.Equal(t, []CampaignAd(nil), res)
}

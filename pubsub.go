package main

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

const pubsubNewAdTopic = "ad-image-processed"
const pubsubNewAdSub = "monetization-new-ad"

var pubsubClient *pubsub.Client = nil

func configurePubsub() error {
	projectID := os.Getenv("GCLOUD_PROJECT")
	ctx := context.Background()

	var err error
	pubsubClient, err = pubsub.NewClient(ctx, projectID, gcpOpts...)
	if err != nil {
		return err
	}

	// Create the subscription if it doesn't exist.
	if exists, err := pubsubClient.Subscription(pubsubNewAdSub).Exists(ctx); err != nil {
		return err
	} else if !exists {
		log.Info("creating pubsub subscription ", pubsubNewAdSub)
		if _, err := pubsubClient.CreateSubscription(context.Background(), pubsubNewAdSub, pubsub.SubscriptionConfig{Topic: pubsubClient.Topic(pubsubNewAdTopic)}); err != nil {
			return err
		}
	}
	return nil
}

func subscribeToNewAd() {
	log.Info("receiving messages from ", pubsubNewAdTopic)
	ctx := context.Background()
	err := pubsubClient.Subscription(pubsubNewAdSub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var ad ScheduledCampaignAd
		if err := json.Unmarshal(msg.Data, &ad); err != nil {
			log.WithField("msg", msg).Errorf("failed to decode message %#v", msg)
			msg.Ack()
			return
		}

		log.Infof("[AD %s] adding new campaign ad", ad.Id)
		if err := addCampaign(ctx, ad); err != nil {
			log.WithField("ad", ad).Errorf("[AD %s] failed to add new campaign ad %v", ad.Id, err)
			msg.Nack()
			return
		}

		msg.Ack()
		log.Infof("[AD %s] added new campaign ad", ad.Id)
	})

	if err != nil {
		log.Fatal("failed to receive messages from pubsub ", err)
	}
}

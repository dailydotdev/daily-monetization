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

const pubsubSegmentTopic = "segment-found"
const pubsubSegmentSub = "monetization-segment-found"

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

	// Create the subscription if it doesn't exist.
	if exists, err := pubsubClient.Subscription(pubsubSegmentSub).Exists(ctx); err != nil {
		return err
	} else if !exists {
		log.Info("creating pubsub subscription ", pubsubSegmentSub)
		if _, err := pubsubClient.CreateSubscription(context.Background(), pubsubSegmentSub, pubsub.SubscriptionConfig{Topic: pubsubClient.Topic(pubsubSegmentTopic)}); err != nil {
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
			log.Errorf("failed to decode message %v", err)
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

type SegmentMessage struct {
	UserId  string
	Segment string
}

func subscribeToSegmentFound() {
	log.Info("receiving messages from ", pubsubSegmentTopic)
	ctx := context.Background()
	err := pubsubClient.Subscription(pubsubSegmentSub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var data SegmentMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Errorf("failed to decode message %v", err)
			msg.Ack()
			return
		}

		if err := updateUserSegment(ctx, data.UserId, data.Segment); err != nil {
			log.WithField("segment", data).Errorf("failed to update user segment %v", err)
			msg.Nack()
			return
		}

		msg.Ack()
		log.WithField("segment", data).Infof("updated user segment")
	})

	if err != nil {
		log.Fatal("failed to receive messages from pubsub ", err)
	}
}

package conference

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

const (
	latestAnnouncementKey = "latestAnnouncement"
	announcementKind      = "Announcement"
)

type Announcement struct {
	Message string
	Time    time.Time
}

func SetLatestAnnouncement(ctx appengine.Context, value string) error {
	setAnnMemcache(ctx, value)
	ann := Announcement{
		Message: value,
		Time:    time.Now(),
	}
	k := datastore.NewKey(ctx, "Announcement", "", 0, nil)
	_, err := datastore.Put(ctx, k, &ann)
	return err
}

func GetLatestAnnouncement(ctx appengine.Context) (string, error) {
	item, err := memcache.Get(ctx, latestAnnouncementKey)
	if err == nil {
		return string(item.Value), nil
	}
	ctx.Infof("memcache miss: %v", err)

	var ann Announcement
	_, err = datastore.NewQuery(announcementKind).
		Order("-Time").
		Limit(1).
		Run(ctx).
		Next(&ann)
	if err != nil {
		// There's no announcements
		if err == datastore.Done {
			return "", nil
		}
		return "", fmt.Errorf("get last announcement from datastore: %v", err)
	}
	SetLatestAnnouncement(ctx, ann.Message)
	return ann.Message, nil
}

func setAnnMemcache(ctx appengine.Context, value string) {
	item := &memcache.Item{
		Key:        latestAnnouncementKey,
		Value:      []byte(value),
		Expiration: 1 * time.Hour,
	}
	if err := memcache.Set(ctx, item); err != nil {
		ctx.Infof("memcache set: %v", err)
	}
}

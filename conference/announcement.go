// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	item := &memcache.Item{
		Key:        latestAnnouncementKey,
		Value:      []byte(value),
		Expiration: 1 * time.Hour,
	}
	if err := memcache.Set(ctx, item); err != nil {
		ctx.Errorf("memcache set: %v", err)
	}

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

	var ann Announcement
	_, err = datastore.NewQuery(announcementKind).
		Order("-Time"). // Order from newer to older
		Limit(1).       // Get only one result at most
		Run(ctx).
		Next(&ann)
	if err == datastore.Done {
		// There's no announcement
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get last announcement from datastore: %v", err)
	}

	SetLatestAnnouncement(ctx, ann.Message)
	return ann.Message, nil
}

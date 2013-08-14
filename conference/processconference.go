package conference

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/taskqueue"
)

func init() {
	http.Handle("/processconference", handler(processConferenceHandler))
	http.Handle("/notifyinterestedusers", handler(notifyInterestedUsersHandler))
}

func confFromKey(ctx appengine.Context, key *datastore.Key) (*Conference, error) {
	var conf Conference
	if err := datastore.Get(ctx, key, &conf); err != nil {
		return nil, fmt.Errorf("get conference: %v", err)
	}
	conf.Key = key.Encode()
	return &conf, nil
}

func confFromKeyStr(ctx appengine.Context, keyStr string) (*Conference, error) {
	key, err := datastore.DecodeKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("wrong key: %v", err)
	}
	return confFromKey(ctx, key)
}

func processConferenceHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	conf, err := confFromKeyStr(ctx, r.FormValue("conf_key"))
	if err != nil {
		return fmt.Errorf("conf from key: %v", err)
	}
	return datastore.RunInTransaction(ctx, func(ctx appengine.Context) error {
		if err := announceConf(ctx, conf); err != nil {
			return fmt.Errorf("announce conf: %v", err)
		}
		if err := postEmailTask(ctx, conf); err != nil {
			return fmt.Errorf("post email task: %v", err)
		}
		if err := sendConfForReview(ctx, conf); err != nil {
			return fmt.Errorf("send conf for review: %v", err)
		}
		return nil
	}, nil)
}

func announceConf(ctx appengine.Context, conf *Conference) error {
	ann := fmt.Sprintf(
		"A new conference has just been scheduled! %s in %s. Don't wait; book now!",
		conf.Name, conf.City)
	SetLatestAnnouncement(ctx, ann)
	return nil
}

func postEmailTask(ctx appengine.Context, conf *Conference) error {
	task := taskqueue.NewPOSTTask(
		"/notifyinterestedusers",
		url.Values{"conf_key": []string{conf.Key}},
	)
	task, err := taskqueue.Add(ctx, task, "")
	if err != nil {
		return fmt.Errorf("add task to default queue: %v", err)
	}
	return nil
}

func sendConfForReview(ctx appengine.Context, conf *Conference) error {
	task := &taskqueue.Task{
		Method:  "PULL",
		Payload: []byte(conf.Key),
	}
	task, err := taskqueue.Add(ctx, task, "review-conference-queue")
	if err != nil {
		return fmt.Errorf("add task to review queue: %v", err)
	}
	return nil
}

func notifyInterestedUsersHandler(w io.Writer, r *http.Request) error {
	ctx := appengine.NewContext(r)
	conf, err := confFromKeyStr(ctx, r.FormValue("conf_key"))
	if err != nil {
		return fmt.Errorf("conf from key: %v", err)
	}

	usrs := []UserProfile{}
	_, err = datastore.NewQuery(UserKind).
		Filter("Topics =", conf.Topic).
		GetAll(ctx, &usrs)
	if err != nil {
		return fmt.Errorf("get interested users: %v", err)
	}

	tos := make([]string, len(usrs))
	for i, usr := range usrs {
		tos[i] = usr.NotifEmail
	}

	msg := &mail.Message{
		Sender:  "campoy@golang.org",
		To:      tos,
		Subject: "Conference you might be interested in: " + conf.Name,
		Body: fmt.Sprintf("Hi! \n We want to let you know that a conference called %s "+
			"has been scheduled to start on %s in %s. We thought you would like "+
			"to know because you are interested in conferences about %s.",
			conf.Name, conf.StartDate.Format("2006-01-02"), conf.City, conf.Topic),
	}
	if err := mail.Send(ctx, msg); err != nil {
		return fmt.Errorf("send mail: %v", err)
	}
	return nil
}

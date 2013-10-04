goconf
======

The Conference Central application manages a set of users and conferences.
Users register providing an email and a list of topics of interest.
When a user creates a new conference on a given topic, all the users interested
in the topic receive a notification via email.

Users can buy tickets for any conference as long as there available tickets.

You can experiment with the application [here](http://go-conf.appspot.com).

It is a complete sample application showing how to use Go on App Engine taking
advantage of the following APIs:

- datastore: for permanent storage of data
- memcache: for temporary storage of announcements
- taskqueue:
  - push queues to perform out-of-request tasks in a robust way
  - pull queues for the review process of conferences
- backends: executing longer tasks as notifying users by email
- mail: to notify users interested in a given topic

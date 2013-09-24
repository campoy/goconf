goconf
======

The Conference Central application allows users to schedule conferences and
register for conferences.

It is a complete sample application showing how to use Go on App Engine taking
advantage of the following APIs:

- datastore: for permanent storage of data
- memcache: for temporary storage of announcements
- taskqueue: to perform out-of-request tasks in a robust way
- backends: executing longer tasks as notifying users by email

Batch scheduler
====

Schedule tasks.

REST
----

### Admin

Listen localhost

`GET /` Splash page

`GET /metrics` Prometheus endpoint

`GET /version` Version

### API

Auth use a JWT token, similar to Hashicorp Vault : https://docs.gitlab.com/ee/ci/examples/authenticating-with-hashicorp-vault/

`schedule` is an UUID

`owner` is `[a-zA-Z-0-9_\-]+` and can't look like an UUID.

`GET /api/schedule` all schedules for admin, my own schedule for a user

`GET /api/schedule/:owner` schedules of this owner

`DELETE /api/schedule/:schedule`

`PUT /api/schedule/:schedule`

`POST /api/schedule` owner is implicit, or explicit if admin creates the schedule.

#### Compose hacked format

```yaml
x-batch:
    start:
    max_wait_time:
    max_execution_time:
    retry:
    every:
    cron:
```

License
-------

3 terms BSD Licence. © 2020 Mathieu Lecarme

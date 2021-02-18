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

`:id` is an UUID

`owner` is `[a-zA-Z-0-9_\-]+` and can't look like an UUID.

`GET /api/task` all schedules for admin, my own schedule for a user

`GET /api/task/:owner` schedules of this owner

`DELETE /api/task/:id`

`PUT /api/task/:id`

`POST /api/task` owner is implicit, or explicit if admin creates the schedule.

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

#### Architecture

`task.Task` is an abstract task to schedule.

Main task implementation is a *docker-compose.yml*.

`task.Action` is an abstract for a running task.

`scheduler.Scheduler` consumes `task.Task`.



License
-------

3 terms BSD Licence. © 2020 Mathieu Lecarme

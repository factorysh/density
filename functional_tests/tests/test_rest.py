import os
import requests
import json
import time
import docker
from pytest import fixture, raises
import datetime
import jwt


@fixture
def session():
    s = requests.Session()
    s.headers.update(
        {
            "Authorization": "Bearer %s"
            % jwt.encode(
                dict(owner="alice", path="/tmp/density/wd/*/volumes/test/answer"),
                os.getenv("AUTH_KEY"),
                algorithm="HS256",
            ),
        }
    )
    return s


def invalid_session():
    s = requests.Session()
    s.headers.update(
        {
            "Authorization": "Bearer %s"
            % jwt.encode(
                dict(owner="alice", path="/no"),
                os.getenv("AUTH_KEY"),
                algorithm="HS256",
            ),
        }
    )
    return s


def test_home(session):
    r = session.get("http://localhost:8042")
    assert r.status_code == 200


def test_tasks(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200
    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
x-batch:
  max_execution_time: 3s
        """
        },
    )
    assert r.status_code == 201


def test_labels(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200
    r = session.post(
        "http://localhost:8042/api/tasks",
        data={
            "labels": json.dumps(
                {"answer": "42", "pika": "chu"},
            )
        },
        files={
            "docker-compose": """
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
x-batch:
  max_execution_time: 3s
        """
        },
    )
    assert r.status_code == 201
    assert r.json()["labels"]["pika"] == "chu"
    r = session.get("http://localhost:8042/api/tasks?pika=chu")
    assert r.status_code == 200
    assert len(r.json()) == 1
    r = session.get("http://localhost:8042/api/tasks?nop=nop")
    assert r.status_code == 200
    assert len(r.json()) == 0


def test_json(session):
    task = {
        "cpu": 2,
        "ram": 128,
        "max_execution_time": "120s",
        "action": {
            "compose": {
                "version": "3",
                "services": {
                    "hello": {"image": "busybox:latest", "command": "echo World"}
                },
            }
        },
    }
    r = session.post("http://localhost:8042/api/tasks", json=task)
    assert r.status_code == 201
    jr = r.json()
    assert jr["retry"] == 0
    assert "id" in jr  # ok, my ID is set
    for k, v in task.items():
        if k != "max_execution_time":  # 120s become 2m0s
            assert jr[k] == v  # information is not altered


def test_prune_on_cancel(session):
    testcases = [
        {"name": "without wait for", "status": 202, "wait_for": False, "flood": 0},
        {"name": "with wait for", "status": 204, "wait_for": True, "flood": 0},
        {
            "name": "without wait for + flood",
            "status": 202,
            "wait_for": False,
            "flood": 3,
            "flood_status": 202,
        },
    ]

    for case in testcases:
        r = session.get("http://localhost:8042/api/tasks")
        assert r.status_code == 200
        r = session.post(
            "http://localhost:8042/api/tasks",
            files={
                "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 2 && echo world'"
x-batch:
    max_execution_time: 3s
"""
            },
        )

        assert r.status_code == 201, "status error in test %s" % case["name"]
        resp = json.loads(r.text)
        id = resp["id"]

        cli = docker.from_env()
        tries = 5
        for tick in range(0, tries):
            try:
                time.sleep(1)
                ct = cli.containers.get("%s_hello_1" % id)
                break
            except Exception as e:
                if tick == tries - 1:
                    raise Exception("Can't find container")
                print(e)

        time.sleep(1)

        base_url = "http://localhost:8042/api/tasks/%s" % id
        url_with_wait = "%s?wait_for" % base_url

        if case["wait_for"]:
            r = session.delete(url_with_wait)
        else:
            r = session.delete(base_url)
        assert r.status_code == case["status"], "status error in test %s" % case["name"]

        if case["flood"] > 0:
            for times in range(0, case["flood"]):
                r = session.delete(base_url)
                assert r.status_code == case["flood_status"], (
                    "flood status error in test %s" % case["name"]
                )

        time.sleep(2)

        with raises(docker.errors.NotFound):
            ct = cli.containers.get("%s_hello_1" % id)


def test_volumes(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200
    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "touch /test/answer"
        volumes:
            - "./test:/test"
x-batch:
    max_execution_time: 2s
"""
        },
    )

    time.sleep(1)

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(2)

    assert os.path.isfile("/tmp/density/wd/{}/volumes/test/answer".format(id))

    r = session.get("http://localhost:8042/api/tasks/%s/volume/test/answer" % id)
    assert r.status_code == 200
    # empty file
    assert r.text == ""

    unauthorized_session = invalid_session()
    r = unauthorized_session.get(
        "http://localhost:8042/api/tasks/%s/volume/test/answer" % id
    )
    assert r.status_code == 401


def test_status(session):
    today = datetime.datetime.now().strftime("%Y/%m/%d")
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200
    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 2 && echo world'"
x-batch:
    max_execution_time: 3s
"""
        },
    )

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(3)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Running"
    assert r.json()["environments"]["DENSITY_STARTED_AT_DATE"] == today


def test_every(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200

    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 2 && echo world'"
x-batch:
    max_execution_time: 3s
    every: 1m
"""
        },
    )

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(2)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["environments"]["DENSITY"] == "true"
    assert r.json()["status"] == "Running"
    assert r.json()["run"]["runner"] == "compose"
    time.sleep(3)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Waiting"


def test_run_history(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200

    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 1 && echo world'"
x-batch:
    max_execution_time: 3s
    every: 2s
"""
        },
    )

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(2)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    time.sleep(4)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    resp = json.loads(r.text)
    assert len(resp["runs"]) == 2
    assert resp["runs"][0]["id"] > resp["runs"][1]["id"]


def test_cron(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200

    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 2 && echo world'"
x-batch:
    max_execution_time: 3s
    cron: "*/1 * * * *"
"""
        },
    )

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(3)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Running"
    time.sleep(3)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Waiting"


def test_cache(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200

    r = session.post(
        "http://localhost:8042/api/tasks",
        files={
            "docker-compose": """
version: '3'
services:
    hello:
        image: "busybox:latest"
        command: "sh -c 'sleep 2 && touch $XDG_CACHE_HOME/test'"
x-batch:
    max_execution_time: 3s
"""
        },
    )

    assert r.status_code == 201
    resp = json.loads(r.text)
    id = resp["id"]

    time.sleep(3)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Running"
    assert os.path.isfile(f"/tmp/density/wd/{id}/cache/test")

import os
import requests
import json
import time
import docker
from pytest import fixture, raises
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


def test_json(session):
    task = {
        "cpu": 2,
        "ram": 128,
        "max_execution_time": "120s",
        "action": {
            "compose": {
                "version": "3",
                "services": {
                    "hello": {"image": "busybox:latest",
                              "command": "echo World"}
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
        {"name": "without wait for",
         "status": 202,
         "wait_for": False,
         "flood": 0},
        {"name": "with wait for",
         "status": 204,
         "wait_for": True,
         "flood": 0},
        {"name": "without wait for + flood",
         "status": 202,
         "wait_for": False,
         "flood": 3,
         "flood_status": 202},
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
                time.sleep(0.5)
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
        assert r.status_code == case["status"], \
            "status error in test %s" % case["name"]

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

    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Waiting"
    time.sleep(2)
    r = session.get("http://localhost:8042/api/task/%s" % id)
    assert r.status_code == 200
    assert r.json()["status"] == "Running"

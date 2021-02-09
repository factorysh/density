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
            % jwt.encode(dict(owner="alice"), "s3cr3t", algorithm="HS256"),
        }
    )
    return s


def test_home(session):
    r = session.get("http://localhost:8042")
    assert r.status_code == 404


def test_schedules(session):
    r = session.get("http://localhost:8042/api/schedules")
    assert r.status_code == 200
    r = session.post(
        "http://localhost:8042/api/schedules",
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
                    "hello": {"image": "busybox:latest", "command": "echo World"}
                },
            }
        },
    }
    r = session.post("http://localhost:8042/api/schedules", json=task)
    assert r.status_code == 201
    jr = r.json()
    assert jr["retry"] == 0
    assert "id" in jr  # ok, my ID is set
    for k, v in task.items():
        if k != "max_execution_time":  # 120s become 2m0s
            assert jr[k] == v  # information is not altered


def test_prune_on_cancel(session):
    testcases = [{"name": "without wait for",
                  "status": 202, "wait_for": False}]

    for case in testcases:
        r = session.get("http://localhost:8042/api/schedules")
        assert r.status_code == 200
        r = session.post(
            "http://localhost:8042/api/schedules",
            files={
                "docker-compose":
                """
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

        url = "http://localhost:8042/api/schedules/%s" % id
        if case["wait_for"]:
            url = "%s?wait_for" % url

        r = session.delete(url)
        assert r.status_code == case["status"], "status error in test %s" % case["name"]
        time.sleep(2)
        with raises(docker.errors.NotFound):
            ct = cli.containers.get("%s_hello_1" % id)

import requests
import json
import jwt


# create fresh session with header token
def session():
    s = requests.Session()
    s.headers.update(
        {
            "Authorization": "Bearer %s"
            % jwt.encode(dict(owner="alice"), "s3cr3t", algorithm="HS256"),
        }
    )
    return s


# run job using provided session
def run(session):
    r = session.get("http://localhost:8042/api/tasks")
    assert r.status_code == 200

    with open("./compose/sitespeed-compose.yml", "r", encoding="utf-8") as compose:
        raw = compose.read()
        print(raw)

    r = session.post(
        "http://localhost:8042/api/tasks",
        files={"docker-compose": raw},
    )

    assert r.status_code == 201


run(session())

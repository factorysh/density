import requests
from pytest import fixture
import jwt


@fixture
def session():
    s = requests.Session()
    s.headers.update({
        'Authorization': 'Bearer %s' % jwt.encode(dict(owner="alice"),
                                                  "s3cr3t", algorithm="HS256"),
    })
    return s


def test_home(session):
    r = session.get('http://localhost:8042')
    assert r.status_code == 404


def test_schedules(session):
    r = session.get('http://localhost:8042/api/schedules')
    assert r.status_code == 200
    r = session.post('http://localhost:8042/api/schedules', files={
        'docker-compose': '''
version: '3'
services:
  hello:
    image: "busybox:latest"
    command: "echo world"
        '''})
    assert r.status_code == 200

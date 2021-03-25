AUTH_KEY:=debug
test: venv
	AUTH_KEY=${AUTH_KEY} ./venv/bin/pytest -s .

start:
	docker network prune -f
	rm -rf /tmp/density
	AUTH_KEY=${AUTH_KEY} DATA_DIR=/tmp/density ../bin/density serve

venv:
	python3 -m venv venv
	./venv/bin/pip install -U pip wheel
	./venv/bin/pip install -r requirements.txt

clean:
	rm -rf venv

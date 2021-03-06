AUTH_KEY:=debug
test: venv
	AUTH_KEY=${AUTH_KEY} ./venv/bin/pytest -s .

start:
	AUTH_KEY=s3cr3t ../bin/density serve

venv:
	python3 -m venv venv
	./venv/bin/pip install -U pip wheel
	./venv/bin/pip install -r requirements.txt

clean:
	rm -rf venv

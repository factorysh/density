test: venv
	./venv/bin/pytest -s .

start:
	AUTH_KEY=s3cr3t ../bin/batch-scheduler

venv:
	python3 -m venv venv
	./venv/bin/pip install -U pip wheel
	./venv/bin/pip install -r requirements.txt

default: venv

output:
	mkdir output

run: output
	./venv/bin/python run.py

venv:
	python3 -m venv venv
	./venv/bin/pip install -U pip wheel
	./venv/bin/pip install -r requirements.txt

build:
	make -C ../

start: build
	AUTH_KEY=s3cr3t make -C ../ run

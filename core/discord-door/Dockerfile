ARG BASE=python:3.12-alpine
FROM ${BASE}
WORKDIR /app
COPY door.py ./
# -I: isolated mode (ignores PYTHON* variables, the user site directory and the current directory).
CMD ["python3", "-I", "-u", "door.py"]

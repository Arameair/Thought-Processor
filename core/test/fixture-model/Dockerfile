# TEST FIXTURE image: a fake llama.cpp server for the clean-host test. Never used for a real install.
ARG BASE=python:3.13-alpine
FROM ${BASE}
COPY fake_llama.py /fixture/fake_llama.py
ENTRYPOINT ["python3", "/fixture/fake_llama.py"]

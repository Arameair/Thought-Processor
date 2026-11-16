# Thought Processor workspace: TASS chat + thought stream + JupyterLab on a shared folder.
ARG BASE=python:3.12-slim
FROM ${BASE}
ARG PIP_FLAGS=""

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    HOME=/workspace/data/home \
    TASS_SHARED=/workspace/shared \
    TASS_DATA=/workspace/data \
    TASS_MIRROR=/shared/mirror

COPY requirements.txt /src/requirements.txt
RUN python3 -m pip install --no-cache-dir ${PIP_FLAGS} -r /src/requirements.txt
COPY tass_workspace /src/tass_workspace
ENV PYTHONPATH=/src

# Unprivileged; compose passes your own UID so ./shared and ./data stay yours.
USER 1000:1000
WORKDIR /workspace/shared
EXPOSE 8790 8888
CMD ["python3", "-m", "tass_workspace"]

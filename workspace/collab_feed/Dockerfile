FROM python:3.12-alpine
WORKDIR /app
COPY observer.py /app/observer.py
USER 1000:1000
ENTRYPOINT ["python3","-u","/app/observer.py"]

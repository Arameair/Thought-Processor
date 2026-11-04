FROM python:3.12-alpine
WORKDIR /app
COPY server.py desk_state.py pulse_cards.py index.html build.html today.html ./
COPY assets/ ./assets/
RUN addgroup -S atrium && adduser -S -G atrium atrium
USER atrium
EXPOSE 8791
CMD ["python3", "-u", "server.py", "--port", "8791", "--backend-port", "8790"]

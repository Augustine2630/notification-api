FROM debian:bookworm-slim
# rsvg-convert + корневые сертификаты
RUN apt-get update && apt-get install -y --no-install-recommends \
      librsvg2-bin \
      ca-certificates \
      curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY ./app /app/app

# каталог для файлов, доступный пользователю nobody
RUN mkdir -p /app/data && chown -R nobody:nogroup /app/data

USER nobody
ENTRYPOINT ["/app/app"]

FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim

WORKDIR /app

ENV UV_PROJECT_ENVIRONMENT=/usr/local

COPY pyproject.toml uv.lock ./
RUN uv sync --no-dev

COPY . .

EXPOSE 3001

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "3001"]

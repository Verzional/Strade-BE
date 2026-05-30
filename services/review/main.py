from contextlib import asynccontextmanager
from collections.abc import AsyncIterator

from fastapi import FastAPI

from app.database import Base, engine
from app.routes.reviews import router as reviews_router


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    Base.metadata.create_all(bind=engine)
    yield


app = FastAPI(title="Review Service", lifespan=lifespan)


@app.get("/health")
def health_check() -> dict[str, str]:
    return {"status": "ok"}


app.include_router(reviews_router)

from datetime import datetime
from typing import Any

from pydantic import BaseModel, ConfigDict, Field


class ReviewRead(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    authorId: str
    author_name: str
    receiverId: str
    receiver_name: str
    rating: int
    description: str
    createdAt: datetime


class ReviewWithUsers(ReviewRead):
    author: dict[str, Any] | None = None
    receiver: dict[str, Any] | None = None
    user_lookup_errors: list[str] = Field(default_factory=list)

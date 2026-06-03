import asyncio
from typing import Annotated
from uuid import UUID

from fastapi import APIRouter, Depends, Query, Header
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.database import get_db
from app.models.review import Review
from app.schemas.review import ReviewWithUsers
from app.services.user_client import UserGatewayClient

router = APIRouter(prefix="/api", tags=["reviews"])


def _valid_user_ids(user_ids: set[str]) -> tuple[list[str], list[str]]:
    valid_ids: list[str] = []
    invalid_ids: list[str] = []
    for user_id in user_ids:
        try:
            valid_ids.append(str(UUID(user_id)))
        except ValueError:
            invalid_ids.append(user_id)
    return valid_ids, invalid_ids


async def _fetch_users_via_gateway(
    user_ids: set[str], 
    auth_token: str | None = None
) -> tuple[dict[str, dict], list[str]]:
    valid_ids, invalid_ids = _valid_user_ids(user_ids)
    if not valid_ids:
        return {}, invalid_ids

    client = UserGatewayClient(auth_token=auth_token)
    users_map = {}
    
    async def fetch_and_map(uid: str):
        user_data, _ = await client.fetch_user(uid)
        if user_data:
            users_map[uid] = user_data

    try:
        await asyncio.gather(*(fetch_and_map(uid) for uid in valid_ids))
    finally:
        await client.close()

    return users_map, invalid_ids


async def _attach_users(
    db: Session,
    reviews: list[Review],
    auth_token: str | None = None
) -> list[ReviewWithUsers]:
    user_ids = {review.authorId for review in reviews} | {review.receiverId for review in reviews}
    
    users, invalid_ids = await _fetch_users_via_gateway({uid for uid in user_ids if uid}, auth_token)

    enriched_reviews: list[ReviewWithUsers] = []
    for review in reviews:
        author = users.get(review.authorId)
        receiver = users.get(review.receiverId)
        errors: list[str] = []
        for user_id, user in ((review.authorId, author), (review.receiverId, receiver)):
            if user_id in invalid_ids:
                errors.append(f"user {user_id}: invalid UUID")
            elif user is None:
                errors.append(f"user {user_id}: not found via gateway")

        enriched_reviews.append(
            ReviewWithUsers.model_validate(review).model_copy(
                update={
                    "author": author,
                    "receiver": receiver,
                    "user_lookup_errors": errors,
                }
            )
        )

    return enriched_reviews


@router.get("/reviews", response_model=list[ReviewWithUsers])
@router.get("/api/v1/reviews", response_model=list[ReviewWithUsers])
@router.get("/api/v1/public-reviews", response_model=list[ReviewWithUsers])
async def fetch_reviews(
    db: Annotated[Session, Depends(get_db)],
    receiverId: Annotated[str | None, Query(description="Filter by review receiver ID")] = None,
    authorization: Annotated[str | None, Header()] = None  # Extract JWT token!
) -> list[ReviewWithUsers]:
    statement = select(Review).order_by(Review.createdAt.desc())
    if receiverId:
        statement = statement.where(Review.receiverId == receiverId)
    reviews = list(db.scalars(statement).all())
    
    # Await the new async attach function
    return await _attach_users(db, reviews, authorization)


@router.get("/reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
@router.get("/api/v1/reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
@router.get("/api/v1/public-reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
async def list_reviews_by_receiver(
    receiver_id: str,
    db: Annotated[Session, Depends(get_db)],
    authorization: Annotated[str | None, Header()] = None  # Extract JWT token!
) -> list[ReviewWithUsers]:
    statement = (
        select(Review)
        .where(Review.receiverId == receiver_id)
        .order_by(Review.createdAt.desc())
    )
    reviews = list(db.scalars(statement).all())
    
    # Await the new async attach function
    return await _attach_users(db, reviews, authorization)
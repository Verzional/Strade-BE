from typing import Annotated
from uuid import UUID

from fastapi import APIRouter, Depends, Query
from sqlalchemy import bindparam, select, text
from sqlalchemy.orm import Session

from app.database import get_db
from app.models.review import Review
from app.schemas.review import ReviewWithUsers

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


def _fetch_users(db: Session, user_ids: set[str]) -> dict[str, dict[str, object]]:
    valid_ids, _ = _valid_user_ids(user_ids)
    if not valid_ids:
        return {}

    statement = text(
        """
        SELECT
            id::text AS id,
            name,
            email,
            image,
            "emailVerified",
            "createdAt",
            "updatedAt"
        FROM users
        WHERE id IN :user_ids
        """
    ).bindparams(bindparam("user_ids", expanding=True))

    rows = db.execute(statement, {"user_ids": valid_ids}).mappings().all()
    return {str(row["id"]): dict(row) for row in rows}


def _attach_users(
    db: Session,
    reviews: list[Review],
) -> list[ReviewWithUsers]:
    user_ids = {review.authorId for review in reviews} | {review.receiverId for review in reviews}
    users = _fetch_users(db, {user_id for user_id in user_ids if user_id})
    _, invalid_ids = _valid_user_ids(user_ids)

    enriched_reviews: list[ReviewWithUsers] = []
    for review in reviews:
        author = users.get(review.authorId)
        receiver = users.get(review.receiverId)
        errors: list[str] = []
        for user_id, user in ((review.authorId, author), (review.receiverId, receiver)):
            if user_id in invalid_ids:
                errors.append(f"user {user_id}: invalid UUID")
            elif user is None:
                errors.append(f"user {user_id}: not found")

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
) -> list[ReviewWithUsers]:
    statement = select(Review).order_by(Review.createdAt.desc())
    if receiverId:
        statement = statement.where(Review.receiverId == receiverId)
    reviews = list(db.scalars(statement).all())
    return _attach_users(db, reviews)


@router.get("/reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
@router.get("/api/v1/reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
@router.get("/api/v1/public-reviews/receiver/{receiver_id}", response_model=list[ReviewWithUsers])
async def list_reviews_by_receiver(
    receiver_id: str,
    db: Annotated[Session, Depends(get_db)],
) -> list[ReviewWithUsers]:
    statement = (
        select(Review)
        .where(Review.receiverId == receiver_id)
        .order_by(Review.createdAt.desc())
    )
    reviews = list(db.scalars(statement).all())
    return _attach_users(db, reviews)

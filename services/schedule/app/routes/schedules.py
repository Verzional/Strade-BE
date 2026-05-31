from typing import Annotated
from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.database import get_db
from app.models.schedule import Schedule
from app.schemas.schedule import ScheduleCreate, ScheduleResponse
from app.services import user_client

router = APIRouter(prefix="/api", tags=["schedules"])

@router.post("/schedules", response_model=ScheduleResponse)
async def create_schedule(
    schedule_in: ScheduleCreate, 
    db: Annotated[Session, Depends(get_db)]
):
    # Gateway request to User Service
    username1 = await user_client.fetch_user_data(schedule_in.userId1)
    username2 = await user_client.fetch_user_data(schedule_in.userId2)

    db_schedule = Schedule(
        time_start=schedule_in.time_start,
        time_end=schedule_in.time_end,
        title=schedule_in.title,
        description=schedule_in.description,
        userId1=schedule_in.userId1,
        username1=username1,
        userId2=schedule_in.userId2,
        username2=username2
    )
    
    db.add(db_schedule)
    db.commit()
    db.refresh(db_schedule)
    return db_schedule

@router.get("/schedules", response_model=list[ScheduleResponse])
async def get_all_schedules(db: Annotated[Session, Depends(get_db)]):
    statement = select(Schedule)
    schedules = db.scalars(statement).all()
    return list(schedules)

@router.get("/schedules/user/{user_id}", response_model=list[ScheduleResponse])
async def get_user_schedules(
    user_id: str, 
    db: Annotated[Session, Depends(get_db)]
):
    statement = select(Schedule).where(
        (Schedule.userId1 == user_id) | (Schedule.userId2 == user_id)
    )
    schedules = db.scalars(statement).all()
    return list(schedules)
from pydantic import BaseModel, ConfigDict
from datetime import datetime
from typing import Optional

class ScheduleCreate(BaseModel):
    time_start: datetime
    time_end: datetime
    title: str
    description: Optional[str] = None
    userId1: str
    userId2: str

class ScheduleResponse(BaseModel):
    id: str
    time_start: datetime
    time_end: datetime
    title: str
    description: Optional[str]
    userId1: str
    username1: str
    userId2: str
    username2: str

    model_config = ConfigDict(from_attributes=True)
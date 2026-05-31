import uuid
from sqlalchemy import Column, String, DateTime
from app.database import Base

def generate_uuid():
    return str(uuid.uuid4())

class Schedule(Base):
    __tablename__ = "schedules"

    id = Column(String, primary_key=True, index=True, default=generate_uuid)
    time_start = Column(DateTime, nullable=False)
    time_end = Column(DateTime, nullable=False)
    title = Column(String, nullable=False)
    description = Column(String, nullable=True)
    
    userId1 = Column(String, nullable=False)
    username1 = Column(String, nullable=False)
    
    userId2 = Column(String, nullable=False)
    username2 = Column(String, nullable=False)
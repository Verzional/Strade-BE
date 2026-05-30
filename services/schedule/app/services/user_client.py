import httpx
from fastapi import HTTPException

# Internal Docker network URL (adjust port/path if your user service differs)
USER_SERVICE_URL = "http://user-service:3000/api/users" 

async def fetch_user_data(user_id: str) -> str:
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{USER_SERVICE_URL}/{user_id}", timeout=5.0)
            
            if response.status_code == 404:
                raise HTTPException(status_code=404, detail=f"User {user_id} not found")
            
            response.raise_for_status()
            data = response.json()
            return data.get("name", "Unknown User")
            
    except httpx.RequestError:
        raise HTTPException(
            status_code=503, 
            detail="User Service is temporarily unavailable. Cannot create schedule."
        )
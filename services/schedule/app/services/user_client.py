import httpx
from fastapi import HTTPException

# Internal Docker network URL (adjust port/path if your user service differs)
GATEWAY_URL = "http://gateway:8080/api/users"

async def fetch_user_data(user_id: str, auth_token: str) -> str:
    try:
        async with httpx.AsyncClient() as client:
            # We must attach the token to the outgoing request so the Gateway lets us in
            headers = {"Authorization": auth_token}
            
            response = await client.get(
                f"{GATEWAY_URL}/{user_id}", 
                headers=headers, 
                timeout=5.0
            )
            
            if response.status_code == 404:
                raise HTTPException(status_code=404, detail=f"User {user_id} not found")
            
            response.raise_for_status()
            data = response.json()
            return data.get("name", "Unknown User")
            
    except httpx.RequestError as e:
        print(f"Gateway connection error: {e}")
        raise HTTPException(
            status_code=503, 
            detail="Gateway/User Service is temporarily unavailable. Cannot create schedule."
        )
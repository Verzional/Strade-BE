from app.database import SessionLocal, engine, Base
from app.models.review import Review

def seed_reviews():
    db = SessionLocal()
    
    print("Clearing old reviews...")
    db.query(Review).delete()
    db.commit()

    print("Seeding new reviews...")
    
    # Your actual ID so these show up on your frontend profile!
    YOUR_ID = "dbc435ac-352b-490c-9e41-8eacdf9f7fa0"
    YOUR_NAME = "Rex Kenny" 

    dummy_reviews = [
        Review(
            authorId="05af7495-1156-4ef6-b58b-71181bfedce",
            author_name="Rex Kenny",
            receiverId=YOUR_ID,
            receiver_name=YOUR_NAME,
            rating=5,
            description="Absolutely amazing developer! Helped me debug the microservices architecture flawlessly."
        ),
        Review(
            authorId=YOUR_ID,
            author_name=YOUR_NAME,
            receiverId="530d9069-9ec1-4213-8248-816621d65ace",
            receiver_name="Kenny Wirasantoso",
            rating=4,
            description="Great collaboration on the Go Gateway. Highly recommend working together."
        )
    ]

    db.add_all(dummy_reviews)
    db.commit()
    print("Successfully seeded reviews!")
    db.close()

if __name__ == "__main__":
    # Ensure tables exist before inserting
    Base.metadata.create_all(bind=engine)
    seed_reviews()
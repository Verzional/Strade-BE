insert dummy review data:
docker compose exec postgres-core psql -U admin -d core_db -c \
"INSERT INTO users (id, name, email, image)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'Dave', 'dave@example.test', 'https://example.test/dave.png'),
  ('22222222-2222-2222-2222-222222222222', 'Alex', 'alex@example.test', 'https://example.test/alex.png')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name, email = EXCLUDED.email, image = EXCLUDED.image, \"updatedAt\" = now();

INSERT INTO reviews (\"authorId\", author_name, \"receiverId\", receiver_name, rating, description)
VALUES ('11111111-1111-1111-1111-111111111111', 'Dave', '22222222-2222-2222-2222-222222222222', 'Alex', 5, 'Great trade partner. Fast response.');"

testcase userservice mati:
curl -sS "http://localhost:8080/api/v1/public-reviews"

expected response:
[{"id":1,"authorId":"11111111-1111-1111-1111-111111111111","author_name":"Dave","receiverId":"22222222-2222-2222-2222-222222222222","receiver_name":"Alex","rating":5,"description":"Great trade partner. Fast response.","createdAt":"2026-05-29T21:21:54.584015Z","author":{"id":"11111111-1111-1111-1111-111111111111","name":"Dave","email":"dave@example.test","image":"https://example.test/dave.png","emailVerified":null},"receiver":{"id":"22222222-2222-2222-2222-222222222222","name":"Alex","email":"alex@example.test","image":"https://example.test/alex.png","emailVerified":null},"user_lookup_errors":[]}]%                                                                                  


- udah connected ke gateway, tapi pas user service mati masih show review

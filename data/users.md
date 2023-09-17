# Register

## users

```bash
# User 1

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "johndoe123",
"password": "P@ssw0rd1",
"email": "johndoe123@example.com",
"phone": "+1234567890",
"role": "user"
}'

# User 2

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "sarahsmith456",
"password": "SecretPass",
"email": "sarahsmith456@example.com",
"phone": "+9876543210",
"role": "user"
}'

# User 3

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "mikebrown789",
"password": "Brownie123",
"email": "mikebrown789@example.com",
"phone": "+5551234567",
"role": "user"
}'

# User 4

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "emilyjones22",
"password": "MyP@ssw0rd",
"email": "emilyjones22@example.com",
"phone": "+3339998888",
"role": "user"
}'

# User 5

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "alexwilson77",
"password": "WilsonPass",
"email": "alexwilson77@example.com",
"phone": "+7771112222",
"role": "user"
}'

# User 6

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "lisaanderson555",
"password": "Anderson123",
"email": "lisaanderson555@example.com",
"phone": "+4446667777",
"role": "user"
}'

# User 7

curl -X POST "http://localhost:3000/api/v1/auth/register" \
 -H "Content-Type: application/json" \
 -d '{
"username": "kevinmartinez88",
"password": "KevMart2022",
"email": "kevinmartinez88@example.com",
"phone": "+1235557777",
"role": "user"
}'
```

## teams

```bash

# Team User 1
curl -X POST "http://localhost:3000/api/v1/auth/register" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "sophieclark44",
       "password": "ClarkPass",
       "email": "sophieclark44@example.com",
       "phone": "+2223334444",
       "role": "team"
   }'

# Team User 2
curl -X POST "http://localhost:3000/api/v1/auth/register" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "brianwright99",
       "password": "Wright1234",
       "email": "brianwright99@example.com",
       "phone": "+9998887777",
       "role": "team"
   }'

# Team User 3
curl -X POST "http://localhost:3000/api/v1/auth/register" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "amymiller333",
       "password": "AmyPass!",
       "email": "amymiller333@example.com",
       "phone": "+6667778888",
       "role": "team"
   }'
```

# Login

## Regular Users (7):

```bash
# User 1 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "johndoe123",
       "password": "P@ssw0rd1"
   }'

# User 2 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "sarahsmith456",
       "password": "SecretPass"
   }'

# User 3 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "mikebrown789",
       "password": "Brownie123"
   }'

# User 4 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "emilyjones22",
       "password": "MyP@ssw0rd"
   }'

# User 5 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "alexwilson77",
       "password": "WilsonPass"
   }'

# User 6 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "lisaanderson555",
       "password": "Anderson123"
   }'

# User 7 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "kevinmartinez88",
       "password": "KevMart2022"
   }'
```

## Team Users (3):

```bash
# Team User 1 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "sophieclark44",
       "password": "ClarkPass"
   }'

# Team User 2 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "brianwright99",
       "password": "Wright1234"
   }'

# Team User 3 Login
curl -X POST "http://localhost:3000/api/v1/auth/login" \
   -H "Content-Type: application/json" \
   -d '{
       "username": "amymiller333",
       "password": "AmyPass!"
   }'
```

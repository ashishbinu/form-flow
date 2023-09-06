1. **Register User Endpoint:**

   - **HTTP Method:** POST
   - **Endpoint:** `/auth/register`
   - **Description:** This endpoint allows a user to register by providing their username, password, email, phone number, and role.

   **Example Request:**

   ```json
   POST /auth/register
   Content-Type: application/json

   {
       "username": "john_doe",
       "password": "password123",
       "email": "john.doe@example.com",
       "phone": "+1234567890",
       "role": "user"
   }
   ```

   **Example Response (Success):**

   ```json
   HTTP/1.1 200 OK
   Content-Type: application/json

   {
       "message": "User registered successfully",
       "user": {
           "id": 1,
           "username": "john_doe",
           "email": "john.doe@example.com",
           "phone": "+1234567890",
           "role": "user"
       }
   }
   ```

   **Example Response (Error - Validation Error):**

   ```json
   HTTP/1.1 400 Bad Request
   Content-Type: application/json

   {
       "error": "Key: 'RegisterRequest.Role' Error:Field validation for 'Role' failed on the 'enum' tag"
   }
   ```

2. **Login User Endpoint:**

   - **HTTP Method:** POST
   - **Endpoint:** `/auth/login`
   - **Description:** This endpoint allows a user to log in by providing their username and password.

   **Example Request:**

   ```json
   POST /auth/login
   Content-Type: application/json

   {
       "username": "john_doe",
       "password": "password123"
   }
   ```

   **Example Response (Success):**

   ```json
   HTTP/1.1 200 OK
   Content-Type: application/json

   {
       "message": "Login successful",
       "user": {
           "id": 1,
           "username": "john_doe",
           "email": "john.doe@example.com",
           "phone": "+1234567890",
           "role": "user"
       },
       "token": "your_access_token"
   }
   ```

   **Example Response (Error - Unauthorized):**

   ```json
   HTTP/1.1 401 Unauthorized
   Content-Type: application/json

   {
       "error": "Invalid credentials"
   }
   ```

3. **Validate Token Endpoint:**

   - **HTTP Method:** GET
   - **Endpoint:** `/auth/validate`
   - **Description:** This endpoint allows you to validate a JWT token obtained after a successful login.

   **Example Request (With Token):**

   ```json
   GET /auth/validate
   Authorization: Bearer your_access_token
   ```

   **Example Response (Success):**

   ```json
   HTTP/1.1 200 OK
   Content-Type: application/json

   {
       "message": "Token is valid",
       "claims": {
           "sub": "1",
           "exp": 1699132263,
           "iat": 1699128663
       }
   }
   ```

   **Example Response (Error - Unauthorized):**

   ```json
   HTTP/1.1 401 Unauthorized
   Content-Type: application/json

   {
       "error": "Token is expired"
   }
   ```

These are the three API endpoints along with example requests and responses in your Go Gin application. Make sure to replace `"your_access_token"` with the actual JWT token when making requests to the `/auth/validate` endpoint.

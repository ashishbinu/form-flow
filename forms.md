# Form Creation

```bash

curl -X POST "http://localhost:3000/api/v1/form/" \
   -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXQiOjE2OTQxODYwNTUsImlhdCI6MTY5NDA5OTY1NSwiaWQiOjEwLCJyb2xlIjoidGVhbSJ9.vK1skReY2gKkuZE3pTdrijSraIA0pYeYLbrQTmIqLAI" \
	-H 'Content-Type: application/json' \
	-d '{
     "title": "Health and Lifestyle Survey",
     "description": "Survey to know health",
     "questions": [
       {
         "type": "text",
         "text": "What is your full name?",
         "required": true
       },
       {
         "type": "text",
         "text": "How old are you?",
         "required": true
       },
       {
         "type": "radio",
         "text": "Do you exercise regularly?",
         "options": ["Yes", "No"]
       },
       {
         "type": "text",
         "text": "How many hours of sleep do you typically get per night?",
         "required": true
       },
       {
         "type": "checkbox",
         "text": "Select your dietary preferences:",
         "options": ["Vegetarian", "Vegan", "Omnivore", "Other"]
       },
       {
         "type": "text",
         "text": "Do you have any known allergies or medical conditions?"
       },
       {
         "type": "text",
         "text": "On average, how many glasses of water do you drink per day?",
         "required": true
       }
     ]
   }'

```

# Form retrieval

```bash
curl -X GET "http://localhost:3000/api/v1/form/1" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXQiOjE2OTQxOTIyMDAsImlhdCI6MTY5NDEwNTgwMCwiaWQiOjksInJvbGUiOiJ1c2VyIn0.lUbX4cJSaATv4-69DFLol6yQeUqO4VIj-QhLLKOhjkA" \
	-H 'Content-Type: application/json'

```

Response

```json
{
  "description": "Survey to know health",
  "id": 1,
  "title": "Health and Lifestyle Survey",
  "questions": [
    {
      "id": 1,
      "required": true,
      "text": "What is your full name?",
      "type": "text"
    },
    {
      "id": 2,
      "required": true,
      "text": "How old are you?",
      "type": "text"
    },
    {
      "id": 3,
      "options": ["Yes", "No"],
      "text": "Do you exercise regularly?",
      "type": "radio"
    },
    {
      "id": 4,
      "required": true,
      "text": "How many hours of sleep do you typically get per night?",
      "type": "text"
    },
    {
      "id": 5,
      "options": ["Vegetarian", "Vegan", "Omnivore", "Other"],
      "text": "Select your dietary preferences:",
      "type": "checkbox"
    },
    {
      "id": 6,
      "text": "Do you have any known allergies or medical conditions?",
      "type": "text"
    },
    {
      "id": 7,
      "required": true,
      "text": "On average, how many glasses of water do you drink per day?",
      "type": "text"
    }
  ]
}
```

# Response creation

- first

```bash
curl -X POST "http://localhost:3000/api/v1/form/responses" \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXQiOjE2OTQxOTI5MDAsImlhdCI6MTY5NDEwNjUwMCwiaWQiOjksInJvbGUiOiJ1c2VyIn0.FDUZVLo0O7ZS6zYhjonRVihfbfTCQZWxP9Y2kpgOCpc" \
	-d '{
    "form_id": 1,
    "answers": [
      {
        "question_id": 1,
        "answer": {
          "type": "text",
          "value": "Alice Smith"
        }
      },
      {
        "question_id": 2,
        "answer": {
          "type": "text",
          "value": "28"
        }
      },
      {
        "question_id": 3,
        "answer": {
          "type": "radio",
          "value": 1
        }
      },
      {
        "question_id": 4,
        "answer": {
          "type": "text",
          "value": "7 hours"
        }
      },
      {
        "question_id": 5,
        "answer": {
          "type": "checkbox",
          "value": [0, 1]
        }
      },
      {
        "question_id": 6,
        "answer": {
          "type": "text",
          "value": "None"
        }
      },
      {
        "question_id": 7,
        "answer": {
          "type": "text",
          "value": "8 glasses"
        }
      }
    ]
  }'

```

- second

```bash
curl -X POST "http://localhost:3000/api/v1/form/responses" \
	-H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXQiOjE2OTQxOTMwMzgsImlhdCI6MTY5NDEwNjYzOCwiaWQiOjcsInJvbGUiOiJ1c2VyIn0.F0ZAXvmhsjtCTmxEMwvoYCLexPmmWi2EPxrNg18mk_M" \
	-d '{
    "form_id": 1,
    "answers": [
      {
        "question_id": 1,
        "answer": {
          "type": "text",
          "value": "Bob Johnson"
        }
      },
      {
        "question_id": 2,
        "answer": {
          "type": "text",
          "value": "35"
        }
      },
      {
        "question_id": 3,
        "answer": {
          "type": "radio",
          "value": 0
        }
      },
      {
        "question_id": 4,
        "answer": {
          "type": "text",
          "value": "6 hours"
        }
      },
      {
        "question_id": 5,
        "answer": {
          "type": "checkbox",
          "value": [2, 3]
        }
      },
      {
        "question_id": 6,
        "answer": {
          "type": "text",
          "value": "None"
        }
      },
      {
        "question_id": 7,
        "answer": {
          "type": "text",
          "value": "6 glasses"
        }
      }
    ]
  }'
```

# Response retrieval

- first

```bash
curl -X GET http://localhost:3000/api/v1/form/responses/1 \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXQiOjE2OTQxODYwNTUsImlhdCI6MTY5NDA5OTY1NSwiaWQiOjEwLCJyb2xlIjoidGVhbSJ9.vK1skReY2gKkuZE3pTdrijSraIA0pYeYLbrQTmIqLAI'
```

Response

```json
{
  "form": {
    "description": "Survey to know health",
    "id": 1,
    "questions": [
      {
        "answer": "Alice Smith",
        "id": 1,
        "text": "What is your full name?",
        "type": "text"
      },
      {
        "answer": "28",
        "id": 2,
        "text": "How old are you?",
        "type": "text"
      },
      {
        "answer": "No",
        "id": 3,
        "options": ["Yes", "No"],
        "text": "Do you exercise regularly?",
        "type": "radio"
      },
      {
        "answer": "7 hours",
        "id": 4,
        "text": "How many hours of sleep do you typically get per night?",
        "type": "text"
      },
      {
        "answer": ["Vegetarian", "Vegan"],
        "id": 5,
        "options": ["Vegetarian", "Vegan", "Omnivore", "Other"],
        "text": "Select your dietary preferences:",
        "type": "checkbox"
      },
      {
        "answer": "None",
        "id": 6,
        "text": "Do you have any known allergies or medical conditions?",
        "type": "text"
      },
      {
        "answer": "8 glasses",
        "id": 7,
        "text": "On average, how many glasses of water do you drink per day?",
        "type": "text"
      }
    ],
    "title": "Health and Lifestyle Survey"
  },
  "id": 1,
  "user_id": 5,
  "submission_time": "2023-09-07T17:08:48.833847Z"
}
```

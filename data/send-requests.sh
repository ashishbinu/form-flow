#!/bin/sh
#

curl -X POST \
	http://localhost:8080/forms \
	-H 'Content-Type: application/json' \
	-d '{
     "title": "Survey",
     "description": "Survey of your personal interests",
     "questions": [
       {
         "type": "text",
         "text": "What is your name?",
         "required": true
       },
       {
         "type": "text",
         "text": "What is your age?",
         "required": true
       },
       {
         "type": "radio",
         "text": "What is your favorite color?",
         "options": ["Red", "Blue", "Green", "Other"]
       },
       {
         "type": "text",
         "text": "Please provide your feedback:",
         "required": true
       },
       {
         "type": "checkbox",
         "text": "Select your interests:",
         "options": ["Sports", "Movies", "Reading", "Travel"]
       }
     ]
   }'
echo
echo '-----------------------------------------------'
sleep 0.1
curl -X GET "http://localhost:8080/forms/1"

echo
echo '-----------------------------------------------'
sleep 0.1
curl -X POST "http://localhost:8080/responses" \
	-H "Content-Type: application/json" \
	-d '{
    "form_id": 1,
    "answers": [
      {
        "question_id": 1,
        "answer": {
          "type": "text",
          "value": "John Doe"
        }
      },
      {
        "question_id": 2,
        "answer": {
          "type": "text",
          "value": "30"
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
          "value": "Great form!"
        }
      },
      {
        "question_id": 5,
        "answer": {
          "type": "checkbox",
          "value": [0, 2]
        }
      }
    ]
  }'

echo
echo '-----------------------------------------------'
sleep 0.1
curl -X GET "http://localhost:8080/responses/1"

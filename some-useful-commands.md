# Commands

- Get access to database in terminal
  ```bash
  docker-compose exec form-db psql -U form_service_man -d form_service_database
  ```
- See the db volumes
  ```bash
    sudo chmod -R a+rwx,go-w ./database/form
  ```

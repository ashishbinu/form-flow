# Backend Architecture Readme

## Introduction

Welcome to the Backend Architecture of our data collection platform. This architecture is designed to ensure scalability, flexibility, and ease of use. It utilizes microservices and a modular plugin system to enhance data ingestion capabilities.

## Architecture Overview

Our system employs a microservices architecture, depicted in the architecture diagram provided. The API Request Flow is as follows:

1. **User sends a request to the API Gateway.**
2. **API Gateway Routes:**
   - Routes the request to the appropriate service based on the request type (Form Service, Auth Service, or Plugin Manager Service).
3. **Authorization Check (for Form Service):**
   - Validates user authorization by forwarding the request to the Auth Service.
   - Routes the request to the Form Service based on validation results.
4. **Event Processing:**
   - Events emitted by the Form Service are queued in RabbitMQ for efficient event handling.
5. **Plugin Management:**
   - Plugin Manager Service efficiently routes events to the appropriate plugins for processing.
6. **Plugin Actions:**
   - Teams can initiate actions on plugins, managed and forwarded by the Plugin Manager Service.

## Technology Stack

Our technology stack includes Golang with the Gin Framework for microservices, PostgreSQL for databases, RabbitMQ for message queuing, and Loki for log aggregation. Grafana is used for monitoring and visualization.

## Setup and Installation

1. Ensure you have `docker` and `docker-compose` installed on your system.
2. Install the Docker logging driver for Loki:

   ```bash
   docker plugin install grafana/loki-docker-driver --alias loki --grant-all-permissions
   ```

3. Run the system using `docker-compose`:

   ```bash
   docker-compose up --build
   ```

Now, your backend architecture is up and running, providing a scalable and modular platform for data collection.

## Accessing Services

- API Gateway: Available at the specified endpoint.
- Form Service: Accessible via the API Gateway at `http://localhost:3000/api/v1/form`.
- Auth Service: Accessible via the API Gateway at `http://localhost:3000/api/v1/auth`.
- Plugin Manager Service: Accessible via the API Gateway at `http://localhost:3000/api/v1/plugins`.
- Grafana: Accessible at `http://localhost:4000`

For detailed API documentation and usage examples, refer to the provided Design Document.

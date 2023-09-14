# Logging

Certainly, I can provide you with a step-by-step implementation guide to add structured logging to your microservices and centralize the logs using Docker Compose. We will use the ELK Stack (Elasticsearch, Logstash, Kibana) along with Filebeat for log shipping. This guide assumes you are using Docker Compose to manage your services.

**Step 1: Set Up Your Microservices for Logging**

In each of your microservices, you will need to configure structured logging using a library like Zap, as discussed earlier. Follow the steps below for each microservice:

1. Add the Zap library to your Go project:

   ```bash
   go get -u go.uber.org/zap
   ```

2. Initialize the logger in your microservice code and use structured logging to log relevant information.

3. Build your microservices into Docker containers with logging enabled. Ensure that logs are written to a file within the container.

**Step 2: Create a Docker Compose File**

Create a Docker Compose file (`docker-compose.yml`) in the root directory of your project to define the ELK Stack and Filebeat services:

```yaml
version: "3"
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.14.1
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node

  logstash:
    image: docker.elastic.co/logstash/logstash:7.14.1
    volumes:
      - ./logstash/config/logstash.yml:/usr/share/logstash/config/logstash.yml
      - ./logstash/pipeline:/usr/share/logstash/pipeline
    ports:
      - "5000:5000"
    depends_on:
      - elasticsearch

  kibana:
    image: docker.elastic.co/kibana/kibana:7.14.1
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

  filebeat:
    image: docker.elastic.co/beats/filebeat:7.14.1
    user: root
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./filebeat/filebeat.yml:/usr/share/filebeat/filebeat.yml
      - ./logs:/var/log/microservices
    depends_on:
      - elasticsearch
    environment:
      - "ELASTICSEARCH_HOST=elasticsearch"
      - "ELASTICSEARCH_PORT=9200"
```

In this Compose file:

- Elasticsearch is used as the storage backend for logs.
- Logstash collects and processes logs from Filebeat and sends them to Elasticsearch.
- Kibana provides a web interface for log visualization.
- Filebeat collects logs from Docker containers and forwards them to Logstash.

**Step 3: Configure Filebeat**

Create a Filebeat configuration file (`filebeat/filebeat.yml`) with the following content:

```yaml
filebeat.inputs:
  - type: docker
    containers.ids:
      - "*"
    processors:
      - add_docker_metadata: ~

output.logstash:
  hosts: ["logstash:5000"]
```

This configuration tells Filebeat to collect logs from all Docker containers and send them to Logstash.

**Step 4: Configure Logstash**

Create a Logstash configuration file (`logstash/config/logstash.yml`) with the following content:

```yaml
http.host: "0.0.0.0"
```

Create a Logstash pipeline configuration file (`logstash/pipeline/logstash.conf`) with a basic configuration:

```conf
input {
  beats {
    port => 5000
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
  }
}
```

This configuration tells Logstash to listen for incoming logs from Filebeat on port 5000 and send them to Elasticsearch.

**Step 5: Start the Services**

Run the following command in your project's root directory to start the services defined in the Docker Compose file:

```bash
docker-compose up -d
```

**Step 6: Integrate Logging in Microservices**

Ensure that your microservices are running and producing logs in a structured format using the Zap library.

**Step 7: Visualize Logs in Kibana**

Access Kibana by visiting `http://localhost:5601` in your web browser. Set up Kibana to create dashboards, visualizations, and alerts based on your structured log data.

With these steps, you'll have centralized structured logging set up for your microservices using the ELK Stack and Filebeat through Docker Compose. Logs from all your microservices will be sent to Elasticsearch, making it easy to search, analyze, and visualize your logs using Kibana.

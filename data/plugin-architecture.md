# Plugin

Imagine a data collection platform that is being used by customers in 50+ countries in over 250 organizations and has powered data collection for over 11 million responses. Its features include team management, multilingual forms, and offline data collection.
Create a plugin manager service which will manager plugins which listens to core platform events and does stuff.
Plugins does various stuff like :

The lifecycle of data collection via this platform does not end with the submission of a response. There is usually some post-submission business logic that the platform needs to support over time. Some real-life examples -

- One of the clients wanted to search for slangs (in local language) for an answer to a text question on the basis of cities (which was the answer to a different MCQ question)

- A market research agency wanted to validate responses coming in against a set of business rules (eg. monthly savings cannot be more than monthly income) and send the response back to the data collector to fix it when the rules generate a flag

- A very common need for organizations is wanting all their data onto Google Sheets, wherein they could connect their CRM, and also generate graphs and charts offered by Sheets out of the box. In such cases, each response to the form becomes a row in the sheet, and questions in the form become columns.

- A recent client partner wanted us to send an SMS to the customer whose details are collected in the response as soon as the ingestion was complete reliably. The content of the SMS consists of details of the customer, which were a part of the answers in the response. This customer

We preempt that with time, more similar use cases will arise, with different “actions” being required once the response hits the primary store/database. We want to solve this problem in such a way that each new use case can just be “plugged in” and does not need an overhaul on the backend. Imagine this as a whole ecosystem for integrations. We want to optimize for latency and having a unified interface acting as a middleman.

So design the plugin system for this microservice backend architecture.

okay so plugins will use

## Plugin service

- plugins can register themselves
- if they are already registered they do below thing
- plugins can signal enable/disable status to plugin manager

```
http://plugin-service/metadata
http://plugin-service/initialize
{
config data
}
i think this should be async (mq here)
http://plugin-service/execute
{
event data
}
this should be sync (so http here)
http://plugin-service/actions
{
action data
}
```

## plugin manager (it has db that stores plugin data)

- get all plugins
- get plugin data
- configure plugin for each team
- team can enable/disable plugins for themselves

may be add message queue between core and manager and service
(see how queues should be setup with one rabbitmq server)

### The plugin manager service:

- It has a db that stores :

  - plugin metadata
    ```json
    {
      "id": "id of the plugin",
      "name": "name of the plugin",
      "description": "description of the plugin",
      // don't show below fields to team
      "url": "url of the plugin service",
      "active_instances": "number of active instances", // it should send to team whether plugin is down or not
      "events": ["onResponse", "onFormCreation"]
    }
    ```
  - plugin global settings for each team
    ```json
    {
      "id": "id of the plugin",
      "team": "id of the team",
      "enabled": true/false
    }
    ```

- So the plugin manager has the api to :
  GET details of all plugins
  GET details of a single plugin
  POST configuration for a plugin for that team (only when it is enabled)
  POST data for a plugin actions (acts as api gateway) (only when it is enabled)
- When an event occurs in the core platform (e.g., onResponse, onFormCreation), it publishes the event data to the appropriate queue or topic based on the event category and team ID.
- It should create new queue for when the plugin registers.
- example code to improve performance of message receiving and enqueueing

```go

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// MessageQueue is a simple message queue for handling Kafka messages.
var MessageQueue chan *kafka.Message

func main() {
	// Initialize the Kafka consumer and topic.
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "your_kafka_broker_address",
		"group.id":          "your_consumer_group_id",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %v", err)
	}

	// Subscribe to the Kafka topic.
	err = consumer.SubscribeTopics([]string{"your_kafka_topic"}, nil)
	if err != nil {
		log.Fatalf("Error subscribing to Kafka topic: %v", err)
	}

	// Initialize the message queue.
	MessageQueue = make(chan *kafka.Message)

	// Start a Goroutine to consume Kafka messages and enqueue them.
	go func() {
		for {
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				// Enqueue the Kafka message.
				MessageQueue <- msg
			} else {
				log.Printf("Error reading Kafka message: %v\n", err)
			}
		}
	}()

	// Initialize the Gin router.
	router := gin.Default()

	// Define an HTTP endpoint to process messages.
	router.POST("/process-message", func(c *gin.Context) {
		message := <-MessageQueue // Dequeue a message from the message queue.

		// Here, you can implement your custom message processing logic.
		// For demonstration purposes, we'll just log the message key and value.
		fmt.Printf("Received Kafka Message: Key: %s, Value: %s\n", string(message.Key), string(message.Value))

		c.JSON(http.StatusOK, gin.H{"message": "Message processed"})
	})

	// Start the Gin server.
	err = router.Run(":8080")
	if err != nil {
		log.Fatalf("Error starting Gin server: %v", err)
	}
}
```

### The individual plugin service has:

- The code has all the details for plugin metadata

  ```json
  {
    "id": "unique id of the plugin",
    "name": "name of the plugin",
    "description": "description of the plugin",
    "url": "url of the plugin service"
  }
  ```

- It has database that stores:
  - plugin configuration data for each team
  - other data that plugin needs to store to function
- It has api so that plugin manager can do:
  - POST configuration for a plugin at http://plugin-service/initialize
  - POST data for a plugin at http://plugin-service/actions (here plugins can expose actions to be called by the plugin manager like export data to excel sheets at http://plugin-service/actions/export)
- It also can receive messages from queue to do tasks based on events
- Also it should do automatic registration of plugin if not already done to plugin manager(It should be http api call i think)
- It should also do send the details if the plugin is working to plugin manager and if the plugin goes down it should send the details to plugin manager(this value increases/decreases the active instances count) . I am not sure if it should be sent as events or http call.

So the flow of events is something like this. events flow from core to plugin manager where the manager db has details on what where the events should be routed. The events should be routed to particular plugin based on the event data if where event_data.team_id == team_id and event_data.event_name == events of enabled plugins of teamid, then send it to that plugin service. How should the design of this communication architecture be?

### How to efficiently route events to plugin-service from plugin manager?

The conditional routing in the Plugin Manager does introduce a potential bottleneck and latency, depending on how it's implemented and the volume of events being processed. To mitigate potential performance issues and delays, you can consider the following strategies:

1. **Efficient Data Structures:** Use efficient data structures and indexing in your Plugin Manager's database to speed up lookups for enabled plugins based on `team_id` and `event_name`. This can help reduce the time it takes to determine which plugins should receive an event.

2. **Parallel Processing:** Implement parallel processing within the Plugin Manager for routing events to plugins. As Kafka topics allow multiple consumers, you can process events concurrently, minimizing delays.

3. **Asynchronous Routing:** Instead of synchronous routing, where the Plugin Manager processes an event immediately upon reception, you can implement an asynchronous routing mechanism. Events can be placed in a queue within the Plugin Manager, and separate worker processes can handle the actual routing to plugins. This decouples event reception from event routing.

4. **Load Balancing:** If the Plugin Manager becomes a performance bottleneck, consider load balancing techniques. Distribute incoming events across multiple instances of the Plugin Manager to share the routing load.

5. **Monitoring and Scaling:** Implement monitoring to track the performance of the Plugin Manager and overall event processing latency. If you notice performance degradation, be prepared to scale the Plugin Manager horizontally to handle increased event traffic.

6. **Caching:** Cache frequently used data in the Plugin Manager to reduce database lookups. This can be particularly useful for configurations that do not change frequently.

7. **Throttling and Prioritization:** Implement throttling mechanisms to manage event traffic during peak loads. Prioritize critical events to ensure they are processed promptly.

8. **Database Optimization:** Regularly optimize the Plugin Manager's database, including indexing, to improve query performance.

9. **Batch Processing:** Consider batch processing for events if applicable. Instead of routing each event individually, you can batch events that share the same `team_id` and `event_name` and process them together.

The specific strategies you adopt will depend on the expected event volume, the complexity of your routing logic, and the performance characteristics of your infrastructure. By carefully implementing and monitoring these strategies, you can minimize delays and ensure efficient event routing in your Plugin Manager.

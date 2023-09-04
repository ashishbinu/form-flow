I am developing data collection platform with microservice architecture , it has a form service which creates forms, get forms, gets response for form. How can I create database for this using docker using the below solution.

--- About storing data for a service ---
Keep each microservice’s persistent data private to that service and accessible only via its API. A service’s transactions only involve its database.

The following diagram shows the structure of this pattern.

The service’s database is effectively part of the implementation of that service. It cannot be accessed directly by other services.

There are a few different ways to keep a service’s persistent data private. You do not need to provision a database server for each service. For example, if you are using a relational database then the options are:

Private-tables-per-service – each service owns a set of tables that must only be accessed by that service
Schema-per-service – each service has a database schema that’s private to that service
Database-server-per-service – each service has it’s own database server.
Private-tables-per-service and schema-per-service have the lowest overhead. Using a schema per service is appealing since it makes ownership clearer. Some high throughput services might need their own database server.

It is a good idea to create barriers that enforce this modularity. You could, for example, assign a different database user id to each service and use a database access control mechanism such as grants. Without some kind of barrier to enforce encapsulation, developers will always be tempted to bypass a service’s API and access it’s data directly.
----------



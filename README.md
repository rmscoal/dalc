# DALC
DALC is a simple distributed calculator service. It distributes it calculation processes to the workers. Meaning the calculation process happens asynchrounously.

There are two components of DALC:
1. Web services
2. Workers

The web service is responsible for creating calculation task and sends it to its workers. The web service is then also responsible to retrieve the calculation result alongside its task's minimal metadata.

The worker on the other hand, is responsible to do the calculations. It receives tasks and calculates the given expression in the task. Once the task has been processed, it updates to the database what the result may be, it could either be SUCCESSFUL or FAILED.

A simple architecture of DALC is the following:

<img src="static/dalc.png" />

## What is this project for?
In short, I am interested in Distributed Systems. Hence, I made dalc for me to apply a simple use case of distributed system (work queues model) using ***RabbitMQ***. On top of that, this project is also for me to learn ***Kubernetes***. Lastly, this project will showcases ***traces monitoring*** in distributed systems.

## Things to do:
- [x] Use RabbitMQ as the message queue 
- [ ] Use Kubernetes to deploy dalc
- [ ] Add Opentelemetry and use Jaeger to display trace
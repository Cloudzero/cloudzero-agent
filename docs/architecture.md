# Application Architecture

This project generates multiple application binaries that can be configured to run as either a Lambda function or a Kubernetes pod. In Lambda, it can handle event bus events and operate behind an API Gateway V1 or V2.

It employs a hexagonal architecture pattern to decouple entry points from the main domain logic, storage components, and event bus components.

## 🏗️ What is Hexagonal Architecture?

Hexagonal Architecture, also known as Ports and Adapters Architecture or Clean Architecture, is a software architecture pattern that promotes loose coupling between the application core (business logic) and external components such as the user interface, database, and external services.

In Hexagonal Architecture, the core of the application is isolated from external components and is accessed through well-defined interfaces or ports. Adapters implement these interfaces to integrate with external components.

## 
# Getting Started
This project simulates how to subscribe a long task progress using Server Sent Event (SSE).

## Setup
---
```
chmod +x start.sh
chmod +x subscribe.sh
```

## Run the app
---
```
go run main.go
```

## Start a task
Start a task with 20 seconds duration.
```
./start.sh 20
```
Will return a task ID
```
1
```

## Subscribe the task
Subcribe the task with a task ID = 1.
```
./subscribe.sh 1
```

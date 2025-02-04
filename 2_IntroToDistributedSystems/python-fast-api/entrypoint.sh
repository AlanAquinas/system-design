#!/bin/bash
# Wait for the database to be ready
echo "Waiting for 5 seconds before starting the app..."
sleep 5

# Start the FastAPI application
uvicorn main:app --host 0.0.0.0 --port 8000
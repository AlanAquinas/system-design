# Use official Python image as a base
FROM python:3.12-slim

RUN apt-get update && apt-get install -y gcc libpq-dev && apt-get clean
# Copy requirements and install dependencies
COPY . .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code


# Expose port 8000 for FastAPI
EXPOSE 8000

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]

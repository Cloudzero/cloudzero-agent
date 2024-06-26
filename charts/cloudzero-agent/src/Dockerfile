# Use a Python image to create the virtual environment and install dependencies
FROM python:3.12-slim AS builder

# Create a directory for the application
WORKDIR /app

# Copy the requirements file and install dependencies in a virtual environment
COPY requirements.txt .
RUN python -m venv venv && \
    . venv/bin/activate && \
    pip install --no-cache-dir -r requirements.txt

# Copy the application code
COPY . .

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /app /app

USER nobody

RUN PATH=/app/venv/bin:$PATH

# Set the entry point to run the application
ENTRYPOINT ["/app/venv/bin/python", "/app/validate.py"]

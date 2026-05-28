# API Gateway Endpoint Registration Guide

This gateway routes public client requests safely to isolated underlying microservices inside our polyglot network. To register your microservice endpoints, **you do not need to modify Go source code.** Instead, you declare your paths via a configuration file.

## How to Register Your Service Endpoints

### Step 1: Create Your Configuration File
Navigate to the `services/gateway/routes/` directory and create a JSON file named after your service (e.g., `review.json`, `skill.json`).

### Step 2: Define Your Routing Logic
Inside your configuration file, map out your service arrays following this specification format:

```json
{
  "routes": [
    {
      "path": "/api/v1/reviews",
      "target_url": "http://review:3001",
      "require_auth": true
    },
    {
      "path": "/api/v1/public-reviews",
      "target_url": "http://review:3001",
      "require_auth": false
    }
  ]
}
# Shiroxy API Documentation

This documentation provides details about the API endpoints present in the Postman collection named 'Shiroxy'.

[![Swagger](https://img.shields.io/badge/Run%20In%20Swagger%20-%23ffgf.svg?style=for-the-badge&logo=swagger&logoColor=black)](http://127.0.0.1:10220/docs/index.html)
[![Postman](https://img.shields.io/badge/Run%20In%20Postman%20-%23F05033.svg?style=for-the-badge&logo=postman&logoColor=white)](https://app.getpostman.com/run-collection/20469355-ce2d4722-8347-459c-9607-aa71842085ce?action=collection%2Ffork&source=rip_markdown&collection-url=entityId%3D20469355-ce2d4722-8347-459c-9607-aa71842085ce%26entityType%3Dcollection%26workspaceId%3Df4746b05-7d6d-406d-9ad5-10af542d204a)

Base URL: `http://127.0.0.1:2210`

## Domains

### register domain

- **Method**: `POST`

- **URL**: `{{LOCAL_BASE_URL}}/domain/register`

- **Request Body**:

```json
{
  "domain": "shikharcode.in",
  "email": "yshikharfzd10@gmail.com",
  "metadata": {
    "name": "Shikhar Yadav"
  }
}
```

- **Response**: `200 OK` (Successful operation)

### Retry SSL

- **Method**: `PATCH`

- **URL**: `{{LOCAL_BASE_URL}}/v1/domain/<domain-name>/retryssl`

- **Response**: `200 OK` (Successful operation)

### Update One Domain

- **Method**: `PATCH`

- **URL**: `{{LOCAL_BASE_URL}}/v1/domain/<domain-name>`

- **Request Body**:

```json
{
  "metadata": {}
}
```

- **Response**: `200 OK` (Successful operation)

### Fetch One Domain

- **Method**: `GET`

- **URL**: `{{LOCAL_BASE_URL}}/v1/domain/<domain-name>`

- **Response**: `200 OK` (Successful operation)

### Remove One Domain

- **Method**: `DELETE`

- **URL**: `{{LOCAL_BASE_URL}}/v1/domain/<domain-name>`

- **Response**: `200 OK` (Successful operation)

## Analytics

### Fetch System Analytics

- **Method**: `GET`

- **URL**: `{{LOCAL_BASE_URL}}/v1/analytics/systems`

- **Response**: `200 OK` (Successful operation)

### Fetch Domain Analytics

- **Method**: `GET`

- **URL**: `{{LOCAL_BASE_URL}}/v1/analytics/domains`

- **Response**: `200 OK` (Successful operation)

## Backends

### Fetch All Backend Servers

- **Method**: `GET`

- **URL**: `{{LOCAL_BASE_URL}}/v1/backends`

- **Response**: `200 OK` (Successful operation)

### Add New Backend Server

- **Method**: `POST`

- **URL**: `{{LOCAL_BASE_URL}}/v1/backends`

- **Request Body**:

```json
{
  "id": "<id>",
  "host": "<host>",
  "port": "<port>",
  "health_url": "<health-url>",
  "tags": ""
}
```

- **Response**: `200 OK` (Successful operation)

### Remove One Backend

- **Method**: `DELETE`

- **URL**: `{{LOCAL_BASE_URL}}/v1/backends/<backend-id>`

- **Response**: `200 OK` (Successful operation)

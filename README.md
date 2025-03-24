# backend-gearfest-interactive-web
to run
- must have `.env` file
- run using docker compose

## api reference
- this api have 2 structure

  - stars
    - name      | string
    - message   | string

  - donation
    - name          | string
    - amount        | float
    - tax_deduction | bool 
    - national_id   | string (nullable)
    - fullname      | string (nullable)
    - email         | string (nullable)
    - bill          | string (nullable, meant to store in base64)

  - if tax_deduction is true, `national_id`, `fullname`, `email`, and `bill` must not be null

- all api failed response will look like this
  - message: error message
  - status: "failed"

#### GET /api/message
  - return
    - success (200)
      - data: 20 stars
      - status: "success"

#### POST /api/message
  - body: stars
  - return
    - success (202)
      - message: "your star will be created soon"
      - status: "accept"

#### POST /api/donate
  - body: donation
  - return
    - success (201)
      - message: "donation created"
      - status: "success"

#### GET /api/top-donate
  - return
    - success (200)
      - data: list of 10 object contain name (name) and donation amount (total_donation)
      - status: "success"

#### GET /api/total-donate
  - return
    - success (200)
      - data: total amount of donation (float)
      - status: "success"

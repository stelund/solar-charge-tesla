# Solar Charge Tesla

Solar Charge Tesla is a control software to read solar panel api:s to determin if enough power is generated to charge
a Tesla car.

Currently SolarEdge's API is supported.

## Design

Solar Charge Tesla runs as a serverless function every 5 minute. When a solar site's power reaches a configured threshold
it will tell the car to start charging.

## State

This project is still a work in progress.

## Environment

The aim for this project is to run on Google's cloud platform with zero costs for single car use.

## License

BSD 3-Clause License

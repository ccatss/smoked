Smoked
======

A Go backend for the [Smokey](https://github.com/kittensaredabest/smokey) looking glass.

Features
--------

- Fast (all compiled code)
- Smaller (no node runtime, only an alpine image)
- Built-in file serving

Upcoming Features
----------

- Automatic HTTPS
- Standalone mode (Run Smoked and Smokey in the same docker container)

Deployment
----------

- Copy `docker-compose.yml` to a local file (no need to clone the repo)
  - Hint: `wget https://raw.githubusercontent.com/ccatss/smoked/main/docker-compose.yml`
- Edit options in `docker-compose.yml`
  - Update `CORS_ORIGIN` to be the domain your looking glass runs on 
  - Features are prefixed with `FEATURE_`
  - Options are prefixed with `FEATURE_TYPE_`
- Run `docker compose up -d` to start the service
  - Use `docker compose pull` to update the image (if desired) 
  - Use `docker compose down` to shut it off
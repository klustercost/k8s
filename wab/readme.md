kudos: https://medium.com/@lorenzouriel/start-guide-to-build-a-meta-whatsapp-bot-with-python-and-fastapi-aee1edfd4132
* To build locally for simple docker tests
    `docker build -t wab/test:v1 .`
* To run locally with docker desktop
    `docker run -d -p 8080:8080 wab/test:v1`
    one can change the ports either in the .env or by settign an env variable

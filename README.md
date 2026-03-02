# aios_lb

Request hedging proxy for aiostreams. Supports comet and stremthru.  
When a request is received at `http://aios_lb:7035/ADDON_TYPE`, it will forward the request to the list of configured instances. Once the first one responds, that response is proxied back to aiostreams.

## Config
```yaml
# config.yml
debug: false
instances:
  - type: # comet | stremthru_torz
    urls:
      - https://instance-1.com
      - https://instance-2.com
```


## Running
Docker:
`docker compose up -d --build`

Manual:
`go run cmd/aios_lb/main.go --config config.yml --listen :7035`


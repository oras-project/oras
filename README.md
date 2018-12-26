# OCI Registry As Storage
`oras` can push/pull any files from/to any registry with OCI support.

If you are using [docker/distribution](https://github.com/docker/distribution), please make sure the version is `2.7.0` or above.

## Push files to remote registry
```
oras push localhost:5000/hello:latest hello.txt
```

## Pull files from remote registry
```
oras pull localhost:5000/hello:latest
```

## Login Credentials
`oras` uses the local docker credential by default. Therefore, please run `docker login` in advance for any private registries.

`oras` also accepts explicit credentials via options. For example,
```
oras pull -u username -p password myregistry.io/myimage:latest
```

## Running in Docker
### Build the image
```
docker build -t oras .
```

### Run on Linux
```
docker run --rm -it -v $(pwd):/workplace oras pull localhost:5000/hello:latest
```

### Run on Windows PowerShell
```
docker run --rm -it -v ${pwd}:/workplace oras pull localhost:5000/hello:latest
```

### Run on Windows Commands
```
docker run --rm -it -v %cd%:/workplace oras pull localhost:5000/hello:latest
```

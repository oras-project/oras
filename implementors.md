# Implementors

ORAS is used to push and pull artifacts to [OCI Artifact][artifacts] supported registries.

The following [Registries Support OCI Artifacts](#registries-supporting-artifacts), with the following [Artifact Types Using ORAS](#artifact-types-using-oras).

See [OCI Artifacts][artifacts] for how to add OCI Artifacts support to your registry, and how to author new artifact types.

## Registries Supporting Artifacts

- [docker/distribution](#docker-distribution) - local/offline verification
- [Azure Container Registry](#azure-container-registry-acr)

## Artifact Types Using ORAS

- [Helm 3 Registries](https://v3.helm.sh/docs/topics/registries/)
- [Singularity](https://sylabs.io/guides/3.1/user-guide/cli/singularity_push.html)

## Docker Distribution

[https://github.com/docker/distribution](https://github.com/docker/distribution) version 2.7+

[docker/distribution](https://github.com/docker/distribution) is a reference implementation of the [OCI distribution-spec][distribution-spec]. Running distribution locally, as a container, provides local/offline verification of ORAS and [OCI Artifacts][artifacts].

### Using a Local, Unauthenticated Container Registry

Run the [docker registry image](https://hub.docker.com/_/registry) locally:

```sh
docker run -it --rm -p 5000:5000 registry
```

This will start a distribution server at `localhost:5000`
*(with wide-open access and no persistence outside of the container)*.

### Using Docker Registry with Authentication

- Create a valid htpasswd file (must use `-B` for bcrypt):

  ```sh
  htpasswd -cB -b auth.htpasswd myuser mypass
  ```

- Start a registry using the password file for authentication:

  ```sh
  docker run -it --rm -p 5000:5000 \
      -v $(pwd)/auth.htpasswd:/etc/docker/registry/auth.htpasswd \
      -e REGISTRY_AUTH="{htpasswd: {realm: localhost, path: /etc/docker/registry/auth.htpasswd}}" \
      registry
  ```

- In a new window, login with `oras`:

  ```sh
  oras login -u myuser -p mypass localhost:5000
  ```

You will notice a new entry for `localhost:5000` appear in `~/.docker/config.json`.

To remove the entry from the credentials file, use `oras logout`:

```sh
oras logout localhost:5000
```

### Using an Insecure Docker Registry

To login to the registry without a certificate, a self-signed certificate, or an unencrypted HTTP connection Docker registry, `oras` supports the `--insecure` flag.

- Create a valid htpasswd file (must use `-B` for bcrypt):

  ```sh
  htpasswd -cB -b auth.htpasswd myuser mypass
  ```

- Start a registry using that file for auth and listen the `0.0.0.0` address:

  ```sh
  docker run -it --rm -p 8443:443 \
      -v $(pwd)/auth.htpasswd:/etc/docker/registry/auth.htpasswd \
      -e REGISTRY_AUTH="{htpasswd: {realm: localhost, path: /etc/docker/registry/auth.htpasswd}}" \
      -e REGISTRY_HTTP_ADDR=0.0.0.0:443 \
      registry
  ```

- In a new window, login with `oras` using the ip address not localhost:

  ```sh
  oras login -u myuser -p mypass --insecure <registry-ip>:8443
  ```

You will notice a new entry for `<registry-ip>:8443` appear in `~/.docker/config.json`.

To remove the entry from the credentials file, use `oras logout`:

```sh
oras logout <registry-ip>:8443
```

### [Azure Container Registry (ACR)](https://aka.ms/acr)

ACR Artifact Documentation: [aka.ms/acr/artifacts](https://aka.ms/acr/artifacts)

- Authenticating with ACR using [Service Principals](https://docs.microsoft.com/azure/container-registry/container-registry-auth-service-principal)

  ```sh
  oras login myregistry.azurecr.io --username $SP_APP_ID --password $SP_PASSWD
  ```

- Authenticating with ACR [using AAD credentials](https://docs.microsoft.com/azure/container-registry/container-registry-authentication) and the [`az cli`](https://docs.microsoft.com/cli/azure/install-azure-cli?view=azure-cli-latest)

  ```sh
  az login
  az acr login --name myregistry
  ```

- Pushing Artifacts to ACR

  ```sh
  oras push myregistry.azurecr.io/samples/artifact:1.0 \
      --manifest-config /dev/null:application/vnd.unknown.config.v1+json \
      ./artifact.txt:application/vnd.unknown.layer.v1+txt
  ```

- Pulling Artifacts from ACR

  ```sh
  oras pull myregistry.azurecr.io/samples/artifact:1.0 \
    --media-type application/vnd.unknown.layer.v1+txt
  ```

## Adding Your Registry or Artifact Type

Do you support Artifacts and ORAS? Please [submit a PR](https://github.com/deislabs/oras/pulls), using similar formatting above. We're happy to promote all usage, as well as feedback.

[artifacts]:            https://github.com/opencontainers/artifacts
[distribution-spec]:    https://github.com/opencontainers/distribution-spec/

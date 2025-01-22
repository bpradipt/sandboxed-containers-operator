# Creating RHCOS Layer

Config files to create RHCOS layered image with Kata.

## With access to OCP cluster

It requires access to an OCP cluster to know the base RHCOS version to use.

Build standard layer with Kata upstream binaries

```sh
make build
```

Build for tdx

```sh
make TEE=tdx build
```

Build for snp

```sh
make TEE=snp build
```

## Without access to OCP cluster

You can also directly build using podman or docker.

- Identify the `rhel-rhcos` layer based on OCP version

  You can get the `rhcos-image` details from the `release.txt` of the specific
  release.
  For example, the `rhel-rhcos` image for OCP 4.16.11 is
  `quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:31feb7503f06db4023350e3d9bb3cfda661cc81ff21cef429770fb12ae878636`
  as can be seen from the
  [release.txt](https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.16.11/release.txt)

- Download the OCP pull secret from console.redhat.com

- Run the following command:


Build for tdx

```sh
podman build --authfile /tmp/pull-secret.json \
   --build-arg OCP_RELEASE_IMAGE=quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:31feb7503f06db4023350e3d9bb3cfda661cc81ff21cef429770fb12ae878636 \
   -t tdx-image -f tdx/Containerfile .
```

Build for snp

```sh
podman build --authfile /tmp/pull-secret.json \
   --build-arg OCP_RELEASE_IMAGE=quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:31feb7503f06db4023350e3d9bb3cfda661cc81ff21cef429770fb12ae878636 \
   -t snp-image -f snp/Containerfile .
```

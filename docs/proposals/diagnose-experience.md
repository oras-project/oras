# Improve ORAS diagnose experience

ORAS currently offers two global options, `--verbose` and `--debug`, which enable users to generate verbose output and logs respectively. These features facilitate both users and developers in inspecting ORAS's performance, interactions with external services and internal systems, and in diagnosing issues by providing a clear picture of the tool’s operations.

Given the diverse roles and scenarios in which ORAS CLI is utilized, we have received feedback from users and developers to improve the diagnostic experience. Enhancing debug logs can significantly benefit ORAS users and developers by making diagnostics clearer and more unambiguous.

This proposal document aims to:

1. Identify the issues associated with the current implementation of the `--verbose` and `--debug` options.
2. Clarify the concepts of different types of output and logs for diagnostic purposes.
3. List the guiding principles to write comprehensive, clear, and conducive debug output and debug logs for effective diagnosis.
4. Propose solutions to improve the diagnostic experience for ORAS CLI users and developers.

## Problem Statement

Specifically, there are existing GitHub issues raised in the ORAS community.

- The user is confused about when to use `--verbose` and `--debug`. See the relevant issue [#1382](https://github.com/oras-project/oras/issues/1382).
- Poor readability of debug logs. No separator lines between request and response information. Users need to add separator lines manually for readability. See the relevant issue [#1382](https://github.com/oras-project/oras/issues/1382).
- Critical information is missing in debug logs. For example, the [error code](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#error-codes) and metadata of the processed resource object (e.g. image manifest) are not displayed.
- The detailed operation information is missing in verbose output. For example, how many and where are objects processed. Less or no verbose output of ORAS commands in some use cases.
- Timestamp of each request and response is missing in debug logs, which is hard to trace historical operation and trace the sequence of events accurately.

## Concepts

At first, the output of ORAS flag `--verbose` and `--debug` should be clarified before restructuring them. 

### Output

There are three types of output in ORAS CLI:

- **Status output**: such as progress information, progress bar in pulling or pushing files.
- **Metadata output**: showing what has been pulled (e.g. filename, digest, etc.) in specified format, such as JSON, text.
- **Content output**: it is to output the raw data obtained from the remote registry server or file system, such as the pulled artifact content save as a file.
- **Error output**: error message are expected to be helpful to troubleshoot where the user has done something wrong and the program is guiding them in the right direction.

The target users of these types of output are standard users. Currently, the output of ORAS `--verbose` flag only exists in oras `pull/push/attach/discover` commands, which prints out detailed status output and metadata output. 

It is intended for end-users who want to observe the detailed file operation when using ORAS. It gives users a comprehensive view of what the tool is doing at every step and how long does it take when push or pull a file.

Since ORAS v1.2.0, progress bar is enabled in `pull/push/attach` by default, thus the ORAS output is already verbose in a terminal.

### Logs

Logs focus on providing technical details for in-depth diagnosing and troubleshooting issues. It is intended for developers or technical users who need to understand the inner workings of the tool. Debug logs are detailed and technical, often including HTTP request and response from interactions between client and server, as well as code-specific information. In general, there are different levels of log including `DEBUG`, `INFO`, `WARNING`, `ERROR`, but only `DEBUG` level log is used in ORAS, which is controlled by the flag `--debug`. 

- **Purpose**: Debug logs are specifically aim to facilitate ORAS developers to diagnose ORAS tool itself. They contain detailed technical information that is useful for troubleshooting problems.
- **Target users**: Primarily intended for developers or technical users who are trying to understand the inner workings of the code and identify the root cause of a possible issue with the tool itself.
- **Content**: Debug logs focus on providing context needed to troubleshoot issues, like variable values, execution paths, error stack traces, and internal states of the application.
- **Level of Detail**: Extremely detailed, providing insights into the application's internal workings and logic, often including low-level details that are essential for debugging.

## Proposals for ORAS CLI

- Deprecate the global flag `--verbose` and only remain `--debug` to avoid ambiguity. Based on the concept above, it is reasonable to continue using `--debug` to enable the output of `DEBUG` level logs in ORAS as it is in ORAS. Meanwhile, This change will make the diagnose experience much more straightforward and less breaking since only ORAS `pull/push/attach/discover` commands have verbose output.
- Introduce a new flag `--detail` to replace the existing global flag `--verbose` of commands like `pull`, `push`, `attach`, and `discover` for detailed output.
- Add separator lines between each request and response for readability.
- Add timestamp of each request and response to the beginning of each request and response.
- Add the response body including [error code](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#error-codes) and the metadata of processed OCI object (e.g. image manifest) to the debug logs
- Define guiding principles and conventions for writing clear and conductive debug logs

### Guiding Principles

Here are the guiding principles to write debug logs.

### 1. **Timestamp Each Log Entry**
- **Precise Timing:** Ensure each log entry has a precise timestamp to trace the sequence of events accurately.
  - Example: `DEBUG: [2023-10-01T12:00:00Z] Starting metadata retrieval for repository oras-demo`

### 2. **Capture API-Specific Details**
- **API Requests:** Log detailed information about API requests made to the registry server, including the HTTP method, endpoint, headers, and body (excluding sensitive information).
  - Example: `DEBUG: [HTTPRequest] POST /v2/oras-demo/blobs/uploads/ Headers: {Content-Length: 524288}`
  
- **API Responses:** Log details about the API responses received, including status codes, headers, and response body (excluding sensitive information).
  - Example: `DEBUG: [HTTPResponse] Status: 201 Created, Headers: {Location: /v2/oras-demo/blobs/uploads/uuid}, Body: {}`

### 3. **Log Before and After Critical Operations**
- **Operation Logs:** Log before performing critical operations and after completing them, including success or failure status.
  - Example: `DEBUG: Starting upload of layer 2 of 3 for repository oras-demo`
  - Example: `DEBUG: Successfully uploaded layer 2 of 3 for repository oras-demo`

- **State Logs:** Log important state information, function names, important variable state, file name and line number before and after key operations or decisions.
  - Example: `DEBUG: Current retry attempt: 1, Max retries: 3`

### 4. **Error and Exception Handling**
- **Catch and Log Exceptions:** Always catch exceptions and log them with relevant context and stack traces.
  - Example: `ERROR: Exception occurred in fetchManifest: Network timeout while accessing /v2/oras-demo/manifests/latest`
  
- **Error Codes:** Include specific error codes to facilitate quick identification and resolution.
  - Example: `ERROR: [ErrorCode: 504] Network timeout while accessing /v2/oras-demo/manifests/latest`

### 5. **Include Actionable Information**
- **Guidance:** Where possible, provide suggestions within logs about potential fixes or next steps for resolving issues.
  - Example: `DEBUG: [Action] Check network connectivity and retry the operation. Error occurred while accessing /v2/oras-demo/manifests/latest`

- **Diagnostic Tips:** Include information that can assist in diagnosing issues like configuration settings, environment variables, or system states.
  - Example: `DEBUG: Current registry URL: https://myregistry.io, Timeout setting:

### 6. **Avoid Logging Sensitive Information**
- **Privacy and Security:** Abstain from logging sensitive information such as passwords, personal data, or security tokens.
  - Example: `DEBUG: Attempting to authenticate user [UserID: usr123]`  (exclude authentication token and password information).

## Investigation on other CLI tools

To make sure the ORAS diagnose functions are natural and easier to use, it worth knowing how diagnose functions work in other popular client tools. 

#### Curl

Curl only has a `--verbose` option to output verbose logs. No `--debug` option.

#### Docker and Podman

Docker provides two options `--debug` and `--log-level`  to control debug logs output within different log levels, such as INFO, DEBUG, WARN, etc. No `--verbose` option. Docker has its own daemon service running in local so its logs might be much more complex.

#### Helm

Helm CLI tool provides a global flag `--debug` to enable verbose output.

## Examples in ORAS

This section lists the current behaviors of ORAS debug logs, proposes the suggested changes to ORAS CLI commands. More examples will be appended below.

### oras copy

Pick the first two requests and responses as examples:

```
oras copy ghcr.io/oras-project/oras:v1.2.0 --to-oci-layout oras-dev:v1.2.0 --debug
```

**Current debug log**

```
[DEBU0000] Request #0
> Request URL: "https://ghcr.io/v2/oras-project/oras/manifests/v1.2.0"
> Request method: "GET"
> Request headers:
   "User-Agent": "oras/1.2.0+Homebrew"
   "Accept": "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.oci.artifact.manifest.v1+json" 
DEBU[0001] Response #0
< Response Status: "401 Unauthorized"
< Response headers:
   "Content-Length": "73"
   "X-Github-Request-Id": "9FC6:30019C:17C06:1C462:66AD0463"
   "Content-Type": "application/json"
   "Www-Authenticate": "Bearer realm=\"https://ghcr.io/token\",service=\"ghcr.io\",scope=\"repository:oras-project/oras:pull\""
   "Date": "Fri, 02 Aug 2024 16:08:04 GMT" 
DEBU[0001] Request #1
> Request URL: "https://ghcr.io/token?scope=repository%3Aoras-project%2Foras%3Apull&service=ghcr.io"
> Request method: "GET"
> Request headers:
   "User-Agent": "oras/1.2.0+Homebrew" 
DEBU[0002] Response #1
< Response Status: "200 OK"
< Response headers:
   "Content-Type": "application/json"
   "Docker-Distribution-Api-Version": "registry/2.0"
   "Date": "Fri, 02 Aug 2024 16:08:05 GMT"
   "Content-Length": "69"
   "X-Github-Request-Id": "9FC6:30019C:17C0D:1C46C:66AD0464" 
```

**Suggested debug logs:**

```
2024-08-02 23:56:02 > msg=Request #0
> Request URL: "https://ghcr.io/v2/oras-project/oras/manifests/v1.2.0"
> Request method: "GET"
> Request headers:
   "User-Agent": "oras/1.2.0+Homebrew"
   "Accept": "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.oci.artifact.manifest.v1+json" 


2024-08-02 23:56:03 <  msg=Response #0
< Response Status: "401 Unauthorized"
< Response headers:
   "Content-Length": "73"
   "X-Github-Request-Id": "9FC6:30019C:17C06:1C462:66AD0463"
   "Content-Type": "application/json"
   "Www-Authenticate": "Bearer realm=\"https://ghcr.io/token\",service=\"ghcr.io\",scope=\"repository:oras-project/oras:pull\""
   "Date": "Fri, 02 Aug 2024 23:56:03 GMT"
< Response body:
{
        "errors": [
            {
                "code": "<UNAUTHORIZED>",
                "message": "<message describing condition>",
                "detail": "<unstructured>"
            },
            ...
        ]
    }

2024-08-02 23:56:02 > msg=Request #1
> Request URL: "https://ghcr.io/token?scope=repository%3Aoras-project%2Foras%3Apull&service=ghcr.io"
> Request method: "GET"
> Request headers:
   "User-Agent": "oras/1.2.0+Homebrew" 



2024-08-02 23:56:03 < msg=Response #1
< Response Status: "200 OK"
< Response headers:
   "Content-Type": "application/json"
   "Docker-Distribution-Api-Version": "registry/2.0"
   "Date": "Fri, 02 Aug 2024 16:08:05 GMT"
   "Content-Length": "69"
   "X-Github-Request-Id": "9FC6:30019C:17C0D:1C46C:66AD0464" 
<  Response body:
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:42c524c48e0672568dbd2842d3a0cb34a415347145ee9fe1c8abaf65e7455b46",
      "size": 1239,
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    },
    ···
}


```

### Show user's environment details

Output the user's running environment details of ORAS such as operating system and architecture information would be helpful to help the ORAS developers locate the issue and reproduce easier. 

For example, the operating system and architecture are supposed to be outputted in `oras version`: 

```bash
$ oras version

ORAS Version:    1.2.0+Homebrew
Go version: go1.22.3
OS/Arch: linux/amd64
```

## Q & A

**Q1:** Is it a common practice to use an environment variable like export ORAS_DEBUG=1 as a global switch for debug logs? What are the Pros and Cons of using this design?

**A:** Per our discussion in the ORAS community meeting, ORAS maintainers agreed to not introduce an additional environment variable as a global switch to enable debug logs since --debug is intuitive enough.

**Q2:**: For the diagnose flag options, why deprecate `--verbose` and remain `--debug` as it is?

**A**: The major reason is that this change avoids overloading the flag `--verbose` and reduce ambiguity in ORAS diagnose experience. Moreover, the `--debug` is consistent with other popular container client tools, such as Helm and Docker. Deprecation of `--verbose` is less breaking than changing behaviors of `--debug`.
# Improve ORAS diagnose experience

ORAS currently offers two global options, `--verbose` and `--debug`, which enable users to generate detailed output and debug logs, respectively. These features facilitate both users and developers in inspecting ORAS's performance, interactions with external services and internal systems, and in diagnosing issues by providing a clear picture of the tool’s operations.

Given the diverse roles and scenarios in which ORAS is utilized, we have received feedback from users and developers on how to improve the diagnostic experience. Enhancing the verbose output and debug logs can significantly benefit ORAS users and developers by making diagnostics clearer and more unambiguous.

This proposal document aims to:

1. Identify the issues associated with the current implementation of the `--verbose` and `--debug` options.
2. Clarify the concepts of verbose output and debug logs.
3. List the guilding principles to write comprehensive, clear, and conducive verbose output and debug logs for effective diagnosis.
4. Propose solutions to improve the diagnostic experience for ORAS users and developers.

## Problem Statement

Specifically, there are exiting GitHub issues raised in the ORAS community.

- The user is confused about when to use `--verbose` and `--debug`. See the relevant issue [#1382](https://github.com/oras-project/oras/issues/1382).
- Poor readability of debug logs. No separator lines between request and response information. Users even manually add separator lines for readability. See the relevant issue [#1382](https://github.com/oras-project/oras/issues/1382).
- Critical information is missing in debug logs. For example, the [error code](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#error-codes) and metadata of the processed resource object (e.g. image manifest) are not displayed.
- The detailed operation information is missing in verbose output. For example, how many resource objects are processed where. Less or none verbose output of ORAS commands in some use cases.
- Timestamp of each request and response is missing in debug logs, which is hard to trace historical performed execution of the tool.

## Concepts

Before re-factoring the log information in the ORAS debug log and [verbose output](https://en.wikipedia.org/wiki/Verbose_mode), it worth clarifying the concepts and differences between verbose output and debug logs.

### Verbose Output 

Verbose output focuses on providing a comprehensive, high-level view of the application's operations. It is intended for end-users who want to observe the detailed normal operation of the tool. It should be human readable and descriptive.

- **Purpose**: Verbose output are typically used to provide highly detailed information about the flow and operations of the application. They give users a comprehensive view of what the tool is doing at every step.
- **Target users**: These are generally intended for users who want to understand the detailed workings of the tool, not necessarily for debugging specific issues.
- **Content**: Verbose output includes lots of informational messages about the application's state, operations performed, configuration details, and more. They provide a broader view, which is helpful for tracing the overall execution of the tool.
- **Level of Detail**: Very detailed, but usually focused on normal operations rather than errors or issues.

### Debug Logs

Debug logs focus on providing technical details for in-depth diagnosing and troubleshooting issues. It is intended for developers or technical users who need to understand the inner workings of the tool. Debug logs are detailed and technical, often including HTTP request and response from interactions between client and server, as well as code-specific information.

- **Purpose**: Debug logs are specifically aimed at helping developers diagnose and fix issues within the application. They contain detailed technical information that is useful for troubleshooting problems.
- **Target users**: Primarily intended for developers or technical users who are trying to understand the inner workings of the code and identify the root cause of issues.
- **Content**: Debug logs focus on providing context needed to troubleshoot issues, like variable values, execution paths, error stack traces, and internal states of the application.
- **Level of Detail**: Extremely detailed, providing insights into the application's internal workings and logic, often including low-level details that are essential for debugging.

## Guiding Principles

Here are the guiding principles to print out debug logs.

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

- **State Logs:** Log important state information before and after key operations or decisions.
  - Example: `DEBUG: Current retry attempt: 1, Max retries: 3`

### 4. **Error and Exception Handling**
- **Catch and Log Exceptions:** Always catch exceptions and log them with relevant context and stack traces.
  - Example: `ERROR: Exception occurred in fetchManifest: Network timeout while accessing /v2/oras-demo/manifests/latest`
  
- **Error Codes:** Include specific error codes to facilitate quick identification and resolution.
  - Example: `ERROR: [ErrorCode: 504] Network timeout while accessing /v2/oras-demo/manifests/latest`

### 5. **Environment and Configuration Details**
- **Environment Information:** Log details about the environment where the tool is running, such as OS version and architecture, tool version, and environment variables (excluding sensitive data).
  - Example: `DEBUG: Running on OS: Ubuntu 20.04, Arch: Linux AMD64 ORAS version: v1.2.0`

- **Configuration Settings:** Include current configuration settings that might affect the execution or behavior of the tool.
  - Example: `DEBUG: Configuration - Registry URL: https://myregistry.io, Timeout: 30s, RetryCount: 3`

### 5. **Include Actionable Information**
- **Guidance:** Where possible, provide suggestions within logs about potential fixes or next steps for resolving issues.
  - Example: `DEBUG: [Action] Check network connectivity and retry the operation. Error occurred while accessing /v2/oras-demo/manifests/latest`

- **Diagnostic Tips:** Include information that can assist in diagnosing issues like configuration settings, environment variables, or system states.
  - Example: `DEBUG: Current registry URL: https://myregistry.io, Timeout setting:

### 6. **Avoid Logging Sensitive Information**
- **Privacy:** Abstain from logging sensitive information such as passwords, personal data, or security tokens.
  - Example: `DEBUG: Attempting to authenticate user [UserID: usr123]` (without password details)

- **Compliance:** Ensure that logs adhere to relevant data protection and privacy regulations.
  - Example: Anonymize or obfuscate sensitive customer data in logs.

## Proposals for ORAS CLI

- Deprecate the global flag "--debug" and only remain "--verbose" to avoid ambiguous usage. 
- Make the debug logs as an optional output controlled by "--verbose" via a parameter. Debug logs should be sent to stderr. 
- Add separator lines between each request and response for readability.
- Add the response body including [error code](https://github.com/opencontainers/distribution-spec/blob/main/spec.md#error-codes) and the metadata of processed resource object (e.g. image manifest) to the debug logs
- Add the detailed operation information to verbose output.
- Add timestamp of each request and response to the beginning of each request and response.

## Investigation

To make sure the ORAS diagnose functions are natural and easier to use, it worth knowing how diagnose functions work in other popular client tools. 

#### Curl

Curl only has a `--verbose` option to output verbose logs. No `--debug` option.

#### Docker

Docker has `--debug` and `--log-level` options to control debug logs output within different log levels, such as INFO, DEBUG, WARN, etc. No `--verbose` option. Docker has its own daemon service running in local so its logs might be much more complex.

#### Kubectl

Kubectl has a command `kubectl logs` to show logs of resource objects such as Pod and container. No separate flags for verbose output and debug logs.

## Examples in ORAS

This section lists the current behaviors of ORAS verbose output and debug logs, proposes the suggested changes for ORAS CLI commands. More examples will be appended below.

### oras copy

**Current debug logs** 

Just pick the first two requests and responses as examples:

```
oras copy ghcr.io/oras-project/oras:v1.2.0 --to-oci-layout oras-dev:v1.2.0 --debug
```

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

**Suggested changes:**

```
$ oras copy ghcr.io/oras-project/oras:v1.2.0 --to-oci-layout oras-dev:v1.2.0 --verbose debug
```

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

## Open Questions

1. Should ORAS applies appropriate log levels to differentiate the log inforamtion? For example:


- **Debug Level:** Reserve the `DEBUG` log level for detailed, technical information meant for developers.
  - Example: `DEBUG: Parsed manifest with 3 layers. Digest: sha256:abcd1234`
  
- **Other Levels:** Use other log levels (Info, Warn, Error) to avoid cluttering debug logs with less vital information.
  - Example (Info): `INFO: Successfully pushed artifact oras-demo:v1`
  - Example (Error): `ERROR: Failed to push artifact oras-demo:v1 due to network timeout`
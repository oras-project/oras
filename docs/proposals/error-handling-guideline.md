# ORAS CLI Error Handling and Message Guideline

This document aims to provide the guidelines for ORAS contributors to improve existing error messages and error handling method as well as the new error output format. It will also provide recommendations and examples for ORAS CLI contributors for how to write friendly and standard error messages, avoid generating inconsistent and ambiguous error messages.

## General guiding principles

A clear and actionable error message is very important when raising an error, so make sure your error message describes clearly what the error is and tells users what they need to do if possible.

First and foremost, make the error messages descriptive and informative. Error messages are expected to be helpful to troubleshoot where the user has done something wrong and the program is guiding them in the right direction. A great error message is recommended to contain the following elements:

- HTTP status code: optional, when the logs are generated from the server side, it's recommended to print the HTTP status code in the error description
- Error description: describe what the error is
- Suggestion: for those errors that have potential solution, print out the recommended solution. Versioned troubleshooting document link is nice to have

Second, when necessary, it is highly suggested for ORAS CLI contributors to provide recommendations for users how to resolve the problems based on the error messages they encountered. Showing descriptive words and straightforward prompt with executable commands as a potential solution is a good practice for error messages.

Third, for unhandled errors you didn't expect the user to run into. For that, have a way to view full traceback information as well as full debug or verbose logs output, and instructions on how to submit a bug.

Fourth, signal-to-noise ratio is crucial. The more irrelevant output you produce, the longer it's going to take the user to figure out what they did wrong. If your program produces multiple errors of the same type, consider grouping them under a single explanatory header instead of printing many similar-looking lines.

Fifth, CLI program termination should follow the standard [Exit Status conventions](https://www.gnu.org/software/libc/manual/html_node/Exit-Status.html) to report execution status information about success or failure. ORAS returns `EXIT_FAILURE` if and only if ORAS reports one or more errors.

Last, error logs can also be useful for post-mortem debugging and can also be written to a file, truncate them occasionally and make sure they don't contain ansi color codes.

## Error output recommendation

### Dos

- Provide full description if the user input does not match what ORAS CLI expected. A full description should include the actual input received from the user and expected input
- Use the capital letter ahead of each line of any error message
- Print human readable error message. If the error message is mainly from the server and varies by different servers, tell users that the error response is from server. This implies that users may need to contact server side for troubleshooting
- Provide specific and actionable prompt message with argument suggestion or show the example usage for reference. (e.g, Instead of showing flag or argument options is missing, please provide available argument options and guide users to "--help" to view more examples)
- If the actionable prompt message is too long to show in the CLI output, consider guide users to ORAS user manual or troubleshooting guide with the versioned permanent link
- If the error message is not enough for troubleshooting, guide users to use "--verbose" to print much more detailed logs
- If server returns an error without any [message or detail](https://github.com/opencontainers/distribution-spec/blob/v1.1.0-rc.3/spec.md#error-codes), such as the example 13 below, consider providing customized and trimmed error logs to make it clearer. The original server logs can be displayed in debug mode

### Don'Ts

- Do not use a formula-like or a programming expression in the error message. (e.g, `json: cannot unmarshal string into Go value of type map[string]map[string]string.`, or `Parameter 'xyz' must conform to the following pattern: '^[-\\w\\._\\(\\)]+$'`)
- Do not use ambiguous expressions which mean nothing to users. (e.g, `Something unexpected happens`, or `Error: accepts 2 arg(s), received 0`)
- Do not print irrelevant error message to make the output noisy. The more irrelevant output you produce, the longer it's going to take the user to figure out what they did wrong.

## How to write friendly error message

### Recommended error message structure

Here is a sample structure of an error message:

```text
{Error|Error response from registry}: {Error description (HTTP status code can be printed out if any)}
[Usage: {Command usage}]
[{Recommended solution}]
```

- HTTP status code is an optional information. Printed out the HTTP status code if the error message is generated from the server side. 
- Command usage is also an optional information but it's recommended to be printed out when user input doesn't follow the standard usage or examples.
- Recommended solution (if any) should follow the general guiding principles described above.

### Examples

Here are some examples of writing error message with helpful prompt actionable information:

#### Example 1: When no reference provided in `oras copy`

Current behavior and output:

```console
$ oras cp
Error: accepts 2 arg(s), received 0
```

Suggested error message:

```console
$ oras cp
Error: "oras copy" requires exactly 2 arguments but received 0.
Usage: oras copy [flags] <from>{:<tag>|@<digest>} <to>[:<tag>[,<tag>][...]]
Please specify 2 arguments as source and destination respectively. Run "oras copy -h" for more options and examples
```

#### Example 2: When user mistakenly uses `tag list` command

Current behavior and output:

```console
$ oras tag list ghcr.io/oras-project/oras
Error: unable to add tag for 'list': invalid reference: missing repository
```

Suggested error message:

```console
$ oras tag list ghcr.io/oras-project/oras
Error: There is no "list" sub-command for "oras tag" command.
Usage: oras tag [flags] <name>{:<tag>|@<digest>} <new_tag> [...]
If you want to list available tags in a repository, use "oras repo tags" 
```

#### Example 3: No tag or digest provided when fetching a manifest

Current behavior and output:

```console
$ oras manifest fetch --oci-layout /tmp/ginkgo1163328512
Error: "/tmp/ginkgo1163328512": no tag or digest when expecting <name:tag|name@digest>
```

Suggested error message:

```console
$ oras manifest fetch --oci-layout /tmp/ginkgo1163328512
Error: "/tmp/ginkgo1163328512": no tag or digest specified
Usage: oras manifest fetch [flags] <name>{:<tag>|@<digest>}
You need to specify an artifact reference in the form of "<name>:<tag>" or "<name>@<digest>". Run "oras manifest fetch -h" for more options and examples 
```

#### Example 4: Push a manifest if no media type provided

Current behavior and output:

```console
$ oras manifest push --oci-layout /sample/images:foobar:mediatype manifest.json
Error: media type is not recognized. 
```

Suggested error message:

```console
$ oras manifest push --oci-layout /sample/images:foobar:mediatype manifest.json
Error: media type is not specified via the flag "--media-type" nor in the manifest.json 
Usage: oras manifest push [flags] <name>[:<tag>[,<tag>][...]|@<digest>] <file>
You need to specify a valid media type in the manifest JSON or via the "--media-type" flag
```

#### Example 5: Attach an artifact if the given option is unknown

Current behavior and output:

```console
$ oras attach --artifact-type oras/test localhost:5000/command/images:foobar --distribution-spec v1.0
Error: unknown distribution specification flag: v1.0
```

Suggested error message:

```console
$ oras attach --artifact-type oras/test localhost:5000/sample/images:foobar --distribution-spec ???
Error: unknown distribution specification flag: "v1.0". 
Available options: v1.1-referrers-api, v1.1-referrers-tag
```

#### Example 6: When attaching, if neither file reference nor annotation is provided

Current behavior and output:

```console
$ oras attach --artifact-type sbom/example localhost:5000/sample/images:foobar
Error: no blob is provided
```

Suggested error message:

```console
$ oras attach --artifact-type sbom/example localhost:5000/sample/images:foobar
Error: neither file nor annotation provided in the command
Usage: oras attach [flags] --artifact-type=<type> <name>{:<tag>|@<digest>} <file>[:<type>] [...]
To attach to an existing artifact, please provide files via argument or annotations via flag "--annotation". Run "oras attach -h" for more options and examples
```

#### Example 7: When pushing files, if the annotation file doesn't match the required format

Current behavior and output:

```console
$ oras push --annotation-file sbom.json ghcr.io/library/alpine:3.9
Error: failed to load annotations from sbom.json: json: cannot unmarshal string into Go value of type map[string]map[string]string. Please refer to the document at https://oras.land/docs/how_to_guides/manifest_annotations
```

Suggested error message:

```console
$ oras push --annotation-file annotation.json ghcr.io/library/alpine:3.9
Error: invalid annotation json file: failed to load annotations from annotation.json.
Annotation file doesn't match the required format. Please refer to the document at https://oras.land/docs/how_to_guides/manifest_annotations
```

#### Example 8: When pushing files, if the annotation value doesn't match the required syntax

Current behavior and output:

```console
$ oras push --annotation "key:value" ghcr.io/library/alpine:3.9
Error: missing key in `--annotation` flag: key:value
```

Suggested error message:

```console
$ oras push --annotation "key:value" ghcr.io/library/alpine:3.9
Error: annotation value doesn't match the required format.
Please use the correct format in the flag: --annotation "key=value"  
```

#### Example 9: When failed to pull files from a public registry

```console
$ oras pull docker.io/nginx:latest
Error: failed to resolve latest: GET "https://registry-1.docker.io/v2/nginx/manifests/latest": response status code 401: unauthorized: authentication required: [map[Action:pull Class: Name:nginx Type:repository]]
```

Suggested error message:

```console
$ oras pull docker.io/nginx:latest
Error response from registry: pull access denied for docker.io/nginx:latest : unauthorized: requested access to the resource is denied
Namespace is missing, do you mean `oras pull docker.io/library/nginx:latest`? 
```

#### Example 10: Neither registry nor OCI image layout provided when pushing a folder 

Current behavior and output:

```console
$ oras push /oras --format json
Error: Head "https:///v2/oras/manifests/sha256:ffa50b27cd0096150c0338779c5090db41ba50d01179d851d68afa50b393c3a3": http: no Host in request URL
```

Suggested error message:

```console
$ oras push /oras --format json
Error: "/oras" is an invalid reference
Usage: oras push [flags] <name>[:<tag>[,<tag>][...]] <file>[:<type>] [...]
Please specify a valid reference in the form of <registry>/<repo>[:tag|@digest]
```

#### Example 11: Push a file or folder that doesn't exist

Current behavior and output:

```console
$ oras push localhost:5000/oras:v1 hello.txt
Error: failed to stat /home/user/hello.txt: stat /home/user/hello.txt: no such file or directory
```

Suggested error message:

```console
$ oras push localhost:5000/oras:v1 hello.txt
Error: /home/user/hello.txt: no such file or directory
```

#### Example 12: Failed to authenticate with registry using an error credential from credential store

Current behavior and output:

```console
$ oras pull localhost:7000/repo:tag --registry-config auth.config
Error: failed to resolve tag: GET "http://localhost:7000/v2/repo/manifests/tag": credential required for basic auth
```

Suggested error message:

```console
$ oras pull localhost:7000/repo:tag --registry-config auth.config
Error: failed to authenticate when attempting to pull: no valid credential found in auth.config
Please check whether the registry credential stored in the authentication file is correct
```

#### Example 13: Failed to resolve the digest with empty error response from registry

Current behavior and output:

```console
oras resolve localhost:7000/command/artifacts:foobar -u t -p 2
WARNING! Using --password via the CLI is insecure. Use --password-stdin.
Error response from registry: <nil>
```

Suggested error message:

```console
oras resolve localhost:7000/command/artifacts:foobar -u t -p 2
WARNING! Using --password via the CLI is insecure. Use --password-stdin.
Error response from registry: recognizable error message not found: failed to resolve digest: HEAD "http://localhost:7000/v2/test/manifests/bar": response status code 401: Unauthorized
Authentication failed. Please verify your login credentials and try again.
```

## Reference

Parts of the content are borrowed from these guidelines.

- [Azure CLI Error Handling Guidelines](https://github.com/Azure/azure-cli/blob/dev/doc/error_handling_guidelines.md)
- [Command Line Interface Guidelines](https://clig.dev/#errors)
- [12 Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
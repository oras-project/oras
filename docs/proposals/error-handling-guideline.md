# ORAS CLI Error Handling and Message Guideline

This document aims to provide the guidelines for ORAS contributors to improve existing error messages and error handling method as well as the new error output format. It will also provide recommendations and examples for ORAS CLI contributors for how to write friendly and standard error messages, avoid generating inconsistent and ambiguous error messages.

## General guiding principles

A clear and actionable error message is very important when raising an error, so make sure your error message describes clearly what the error is and tells users what they need to do if possible.

First and foremost, make the error messages descriptive and informative. Error messages are expected to be helpful to troubleshoot where the user has done something wrong and the program is guiding them in the right direction. A great error message is recommended to contain the following elements:

- Error code: optional, when the logs are generated from the server side
- Error description: describe what the error is
- Suggestion: How to fix the error. Versioned troubleshooting document link is nice to have.

Second, when necessary, it is highly suggested for ORAS CLI contributors to provide recommendations for users how to resolve the problems based on the error messages they encountered. Showing descriptive words and straightforward prompt with executable commands as a potential solution is a good practice for error messages.

Third, for unhandled errors you didn’t expect the user to run into. For that, have a way to view full traceback information as well as full debug or verbose logs output, and instructions on how to submit a bug.

Fourth, signal-to-noise ratio is crucial. The more irrelevant output you produce, the longer it’s going to take the user to figure out what they did wrong. If your program produces multiple errors of the same type, consider grouping them under a single explanatory header instead of printing many similar-looking lines.

Fifth, CLI program termination should follow the standard [Exit Status conventions](https://www.gnu.org/software/libc/manual/html_node/Exit-Status.html) to report execution status information about success or failure. 

Last, error logs can also be useful for post-mortem debugging but make sure they have timestamps, truncate them occasionally so they don’t eat up space on disk, and make sure they don’t contain ansi color codes. Thereby, error logs can be written to a file.

## Error output recommendation

### Dos

- Provide full description if the user input does not match what ORAS CLI expected. A full description should include the actual input received from the user and expected input
- Use the capital letter ahead of each line of any error message
- Print human readable error message. If the error message is mainly from the server and varies by different servers, tell users that the error response is from server. This implies that users may need to contact server side for troubleshooting.
- Provide specific and actionable prompt message with argument suggestion or show the example usage for reference. (e.g, Instead of showing flag or argument options is missing, please provide available argument options and guide users to "--help" to view more examples)
- If the actionable prompt message is too long to show in the CLI output, consider guide users to ORAS user guide or troubleshooting guide with the permanent link.
- If the error message is not enough for troubleshooting, guide users to use "--verbose" to print much more detailed logs

### Don'Ts

- Do not use a formula-like or a programming expression in the error message. (e.g, `json: cannot unmarshal string into Go value of type map[string]map[string]string.`, or `Parameter 'xyz' must conform to the following pattern: '^[-\\w\\._\\(\\)]+$'`)
- Do not use ambiguous expressions which mean nothing to users. (e.g, `Something unexpected happens`, or `Error: accepts 2 arg(s), received 0`)
- Do not print irrelevant error message to make the output noisy. The more irrelevant output you produce, the longer it’s going to take the user to figure out what they did wrong.

## How to write friendly error message

### Recommended error message structure

Here is a sample structure of an error message:

```text
Error:[Status code]: [Error description] 
Usage: [Command usage]
[Recommended solution]
```

- Status code is an optional information. If the error message is generated from the server side, it may include error code or [warn code](https://www.rfc-editor.org/rfc/rfc7234#section-5.5). It could be printed out alongside the error description.
- Command usage is also an optional information but it's recommended to be printed out when user input doesn't follow the standard usage or examples. 
- Recommended solution is required and should follow the general guiding principles described above.

### Examples

Here are some examples of writing error message with helpful prompt actionable information:

#### Example 1: when no reference provided in `oras copy`

Current behavior and output:

```
$ oras cp
Error: accepts 2 arg(s), received 0
```

Suggested error message:

```
$ oras cp
Error: "oras copy" requires exactly 2 arguments.
Usage: oras copy [flags] <from>{:<tag>|@<digest>} <to>[:<tag>[,<tag>][...]]
Copy artifacts from one target to another. Run "oras copy -h" for more options and examples
```

#### Example 2: when reference does not match with the expected format

Current behavior and output:

```
$ oras tag list ghcr.io/oras-project/oras
Error: unable to add tag for 'list': invalid reference: missing repository
```

Suggested error message:

```
$ oras tag list ghcr.io/oras-project/oras
Error: There is no "list" sub-command for "oras tag" command.
Usage: oras tag [flags] <name>{:<tag>|@<digest>} <new_tag> [...]
Tag a manifest in a registry or an OCI image layout. Run "oras tag -h" for more options and examples
```

#### Example 3: When fetching a manifest if no manifest tag or digest is provided

Current behavior and output:

```
$ oras manifest fetch --oci-layout /tmp/ginkgo1163328512
Error: "/tmp/ginkgo1163328512": no tag or digest when expecting <name:tag|name@digest>
```

Suggested error message:

```
$ oras manifest fetch --oci-layout /tmp/ginkgo1163328512
Error: "/tmp/ginkgo1163328512": no tag or digest specified
Usage: oras manifest fetch [flags] <name>{:<tag>|@<digest>}
Fetch manifest of the target artifact. Run "oras manifest fetch -h" for more options and examples 
```

#### Example 4: push a manifest if no media type flag provided

Current behavior and output:

```
$ oras manifest push --oci-layout /sample/images:foobar:mediatype
Error: media type is not recognized. 
```

Suggested error message:

```
$ oras manifest push --oci-layout /sample/images:foobar:mediatype
Error: media type is not recognized. 
Usage: oras manifest push [flags] <name>[:<tag>[,<tag>][...]|@<digest>] <file>
Specify an valid media type with the "--media-type" flag.
```

#### Example 5: attach an artifact if the given option is unknown

Current behavior and output:

```
$ oras attach --artifact-type oras/test localhost:5000/command/images:foobar --distribution-spec v1.0
Error: unknown distribution specification flag: v1.0
```

Suggested error message:

```
$ oras attach --artifact-type oras/test localhost:5000/sample/images:foobar --distribution-spec ???
Error: unknown distribution specification flag: "v1.0". 
Available options: v1.1-referrers-api, v1.1-referrers-tag
```

#### Example 6: when attaching an file, if no file reference is provided

Current behavior and output:

```
$ oras attach --artifact-type sbom/example localhost:5000/sample/images:foobar
Error: no blob is provided
```

Suggested error message:

```
$ oras attach --artifact-type sbom/example localhost:5000/sample/images:foobar
Error: failed to attach a file. No file provided in the command.
Usage: oras attach [flags] --artifact-type=<type> <name>{:<tag>|@<digest>} <file>[:<type>] [...]
To attach files to an existing artifact, try "oras attach --artifact-type sbom/example localhost:5000/sample/images:foobar sample.json". Run "oras attach -h" for more options and examples
```

#### Example 7: When pushing files, if the annotation file doesn't match the required format

Current behavior and output:

```
$ oras push --annotation-file sbom.json ghcr.io/library/alpine:3.9
Error: failed to load annotations from sbom.json: json: cannot unmarshal string into Go value of type map[string]map[string]string. Please refer to the document at https://oras.land/docs/how_to_guides/manifest_annotations.
```

Suggested error message:

```
$ oras push --annotation-file annotation.json ghcr.io/library/alpine:3.9
Error: failed to load annotations from annotation.json: annotation file or syntax doesn't match the required format or syntax
Please refer to the document at https://oras.land/docs/how_to_guides/manifest_annotations.
```

#### Example 8: When pushing files, if the annotation value doesn't match the required syntax

Current behavior and output:

```
$ oras push --annotation "key:value" ghcr.io/library/alpine:3.9
Error: missing key in `--annotation` flag: key:value
```

Suggested error message:

```
$ oras push --annotation "key:value" ghcr.io/library/alpine:3.9
Error: annotation value doesn't match the required format.
Try `oras push --annotation "key=value" ghcr.io/library/alpine:3.9`  
```

#### Example 9: when failed to pull files from a public registry

```
$ oras pull docker.io/nginx:latest
Error: failed to resolve latest: GET "https://registry-1.docker.io/v2/nginx/manifests/latest": response status code 401: unauthorized: authentication required: [map[Action:pull Class: Name:nginx Type:repository]]
```

Suggested error message:

```
$ oras pull docker.io/nginx:latest
Error: failed to resolve the resource from server: pull access denied for ghcr.io/nginx:latest : response status code 401: unauthorized: requested access to the resource is denied
Namespace is missing, do you mean `oras pull docker.io/library/nginx:latest`? 
```

## Reference

Parts of the content are borrowed from these guidelines.

- [Azure CLI Error Handling Guidelines](https://github.com/Azure/azure-cli/blob/dev/doc/error_handling_guidelines.md)
- [Command Line Interface Guidelines](https://clig.dev/#errors)
- [12 Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
# ORAS CLI Error Handling Guidelines

This document aims to provide the guidelines for ORAS contributors to improve existing error messages and error handling method as well as the new error output format. It will also provide recommendations for ORAS CLI contributors for how to write friendly and standard error messages and avoid inconsistent error handling & messages.

## General guiding principles

A clear and actionable error message is very important when raising an error, so make sure your error message describes clearly what the error is and tells users what they need to do if possible.

First and foremost, make the error messages descriptive and informative. Error messages are expected to be helpful to troubleshoot where the user has done something wrong and the program is guiding them in the right direction. A great error message is recommended to contain the following:

- Error code
- Error title
- Error description 
- How to fix the error (Optional)
- URL for more information (Optional, it can be a TSG document link)

Second, when necessary, it is highly suggested for ORAS CLI contributors to provide recommendations for users how to resolve the problems based on the error messages they encountered. Showing descriptive words and straightforward prompt with executable commands as a potential solution is a good practice for error messages.

Third, for unhandled errors you didn’t expect the user to run into. For that, have a way to view full traceback information as well as full debug or verbose logs output, and instructions on how to submit a bug.

Forth, signal-to-noise ratio is crucial. The more irrelevant output you produce, the longer it’s going to take the user to figure out what they did wrong. If your program produces multiple errors of the same type, consider grouping them under a single explanatory header instead of printing many similar-looking lines.

Last, error logs can also be useful for post-mortem debugging but make sure they have timestamps, truncate them occasionally so they don’t eat up space on disk, and make sure they don’t contain ansi color codes. Thereby, error logs can be written to a file.

## How to write error message

Here is an sample structure of an error message:

```text
Error: [Error description] : [Error code] ：[Error title]
Help: [Recommended solution], [TSG]
```

See an example:

```
$ oras pull docker.io/nginx:latest
Error: failed to resolve the resource: pull access denied for docker.io/nginx:latest : response status code 401: unauthorized: requested access to the resource is denied
Help: repository does not exist or namespace is missing, or may require 'oras login'. Consider trying "docker.io/library/nginx:latest".
```

## Error types

TBD

## Error Recommendation

### Dos

### Don'Ts
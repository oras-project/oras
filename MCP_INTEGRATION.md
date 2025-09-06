# ORAS MCP Server Integration

ORAS now includes support for the Model Context Protocol (MCP), allowing it to be used as an MCP server with AI assistants like Claude Desktop and IDEs like VSCode.

## What is MCP?

The Model Context Protocol (MCP) enables AI assistants to interact with external tools and data sources. By configuring ORAS as an MCP server, AI assistants can use ORAS commands to help with container registry operations.

## Available Commands

The MCP integration exposes the following safe, read-only ORAS commands:
- `discover` - Discover referrers of a manifest
- `pull` - Pull files from a registry
- `resolve` - Resolve digest of artifacts
- `version` - Show version information

## Setup

### Claude Desktop

To enable ORAS as an MCP server in Claude Desktop:

```bash
oras mcp claude enable
```

This will automatically configure your Claude Desktop to use ORAS as an MCP server.

### VSCode

To enable ORAS as an MCP server in VSCode:

```bash
oras mcp vscode enable
```

For workspace-specific configuration:
```bash
oras mcp vscode enable --workspace
```

## Usage

### Starting the MCP Server

To manually start the MCP server:

```bash
oras mcp start
```

### Exporting Tools

To see what tools are available:

```bash
oras mcp tools
```

This generates a `mcp-tools.json` file containing the tool definitions.

### Disabling

To remove ORAS from Claude Desktop configuration:
```bash
oras mcp claude disable
```

To remove from VSCode configuration:
```bash
oras mcp vscode disable
```

## Examples

Once configured, you can ask your AI assistant to help with tasks like:

- "Show me the version of ORAS"
- "Discover referrers for a manifest"
- "Pull an artifact from a registry"
- "Resolve the digest of an image"

The AI assistant will use the appropriate ORAS commands and provide formatted responses.

## Security

The MCP integration only exposes safe, read-only operations. Destructive operations like delete, push, or authentication commands are not available through the MCP interface to ensure safety when used with AI assistants.
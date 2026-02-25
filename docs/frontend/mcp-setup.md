# MCP Setup for API Schema

This guide explains how to configure the Model Context Protocol (MCP) server for the Doc Engine API schema, enabling AI agents to efficiently query the OpenAPI specification.

## Why Use MCP?

The Swagger/OpenAPI specification for this project is ~274KB (JSON) / ~147KB (YAML). Loading the entire file into an LLM's context window:

- Consumes a significant portion of the token budget
- Increases costs per request
- Reduces available context for actual work

**MCP solves this** by exposing the API schema through on-demand tools. The LLM queries only what it needs (specific endpoints, schemas, parameters) instead of loading everything.

## Prerequisites

- Node.js 18+ installed
- npm/npx available in PATH

## Configuration by Agent

### Claude Code (CLI)

Add the MCP server using the CLI:

```bash
# Local scope (current project only, stored in ~/.claude.json)
claude mcp add doc-engine-api -- npx -y mcp-openapi-schema /path/to/core/docs/swagger.yaml

# Project scope (shared via .mcp.json in repo)
claude mcp add doc-engine-api -s project -- npx -y mcp-openapi-schema ./core/docs/swagger.yaml

# User scope (available in all projects)
claude mcp add doc-engine-api -s user -- npx -y mcp-openapi-schema /absolute/path/to/swagger.yaml
```

**Verify installation:**

```bash
claude mcp list
claude mcp get doc-engine-api
```

**Remove if needed:**

```bash
claude mcp remove doc-engine-api
```

---

### OpenAI Codex

Edit `~/.codex/config.toml`:

```toml
[mcp_servers.doc-engine-api]
command = "npx"
args = ["-y", "mcp-openapi-schema", "/path/to/core/docs/swagger.yaml"]

# Optional settings
startup_timeout_sec = 30
tool_timeout_sec = 60
```

**Via CLI:**

```bash
codex mcp add doc-engine-api -- npx -y mcp-openapi-schema /path/to/core/docs/swagger.yaml
```

---

### Gemini CLI

Edit `~/.gemini/settings.json` (global) or `.gemini/settings.json` (project):

```json
{
  "mcpServers": {
    "doc-engine-api": {
      "command": "npx",
      "args": ["-y", "mcp-openapi-schema", "/path/to/core/docs/swagger.yaml"]
    }
  }
}
```

> **Note:** Restart Gemini CLI after modifying the configuration.

---

## Available Tools

Once configured, the following tools are available to the LLM:

| Tool                    | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `list-endpoints`        | Lists all API paths with HTTP methods and summaries           |
| `get-endpoint`          | Gets detailed information about a specific endpoint           |
| `get-request-body`      | Gets the request body schema for an endpoint                  |
| `get-response-schema`   | Gets the response schema by endpoint, method, and status code |
| `get-path-parameters`   | Gets the parameters for a specific path                       |
| `list-components`       | Lists all schema components (DTOs, responses, parameters)     |
| `get-component`         | Gets the detailed definition for a specific component         |
| `list-security-schemes` | Lists available security/authentication schemes               |
| `get-examples`          | Gets examples for a component or endpoint                     |
| `search-schema`         | Searches across paths, operations, and schemas                |

## Usage Examples

### Finding Endpoints

Ask the LLM:

> "List all endpoints related to templates"

The LLM will use `search-schema` or `list-endpoints` to find relevant paths.

### Understanding a Request

Ask the LLM:

> "What parameters does POST /api/v1/templates require?"

The LLM will use `get-endpoint`, `get-request-body`, and `get-path-parameters`.

### Exploring DTOs

Ask the LLM:

> "Show me the structure of the TemplateResponse DTO"

The LLM will use `get-component` to retrieve the schema definition.

### Generating Code

Ask the LLM:

> "Generate a Go client for the workspace endpoints"

The LLM will use multiple tools to understand the endpoints and generate accurate code.

## Troubleshooting

### Server Not Connecting

1. Verify npx is in PATH:

   ```bash
   which npx
   ```

2. Test the MCP server manually:

   ```bash
   npx -y mcp-openapi-schema /path/to/swagger.yaml --help
   ```

3. Check the swagger file exists and is valid:

   ```bash
   ls -la /path/to/swagger.yaml
   ```

### Tools Not Available

1. For Claude Code, verify with:

   ```bash
   claude mcp list
   ```

2. Ensure the server shows as "Connected"

3. Restart the agent/CLI after configuration changes

### Slow Responses

The first invocation may be slow due to npx downloading the package. Subsequent calls are faster as the package is cached.

## References

- [mcp-openapi-schema](https://github.com/hannesj/mcp-openapi-schema) - The MCP server package
- [Claude Code MCP Docs](https://code.claude.com/docs/en/mcp)
- [OpenAI Codex MCP](https://developers.openai.com/codex/mcp/)
- [Gemini CLI MCP](https://geminicli.com/docs/tools/mcp-server/)
- [Model Context Protocol](https://modelcontextprotocol.io/) - Official MCP specification

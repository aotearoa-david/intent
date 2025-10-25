# Technology Choices

1. For mocking AWS use localstack.
2. For backend code, use Go lang and https://github.com/mark3labs/mcp-go for Model Context Protocol implementation and POSTGRES database.
3. The primary artifact is the API Definition, use Open API 3.1 as per https://swagger.io/specification/
4. It is important to define any interface or MCP protocol details in README.md
5. Unit tests are required for all backend functions.
6. For frontent code, use Typescript and React.
7. Always document the architecture using mermaid to update the c4model system context diagram and the c4model container diagram

# VS Code Extension Publishing Workflow

This workflow automatically publishes the Temporal VS Code debugger extension to the VS Code marketplace.

## Setup

### 1. Create a Personal Access Token (PAT)

1. Go to [Visual Studio Code Marketplace](https://marketplace.visualstudio.com/)
2. Sign in with your Microsoft account
3. Go to your publisher page (temporal-technologies)
4. Create a Personal Access Token (PAT) with publish permissions

### 2. Add GitHub Secret

1. Go to your GitHub repository settings
2. Navigate to "Secrets and variables" → "Actions"
3. Add a new repository secret:
   - **Name**: `VSCE_PAT`
   - **Value**: Your VS Code marketplace PAT

## Usage

### Automatic Publishing (Recommended)

The workflow automatically publishes when you:

1. **Push a tag** starting with 'v' (e.g., `v1.0.0`, `v2.1.3`)
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Push to main branch** (will package but not publish)

### Manual Publishing

1. Go to the "Actions" tab in your GitHub repository
2. Select "Publish VS Code Extension" workflow
3. Click "Run workflow"
4. Enter the version number (e.g., `1.0.0`)
5. Click "Run workflow"

## Workflow Steps

1. **Checkout code** - Gets the latest code
2. **Setup Node.js** - Installs Node.js 18 with npm caching
3. **Install dependencies** - Runs `npm ci` in the extension directory
4. **Run linting** - Executes ESLint and Prettier checks
5. **Run tests** - Executes the test suite
6. **Build extension** - Compiles TypeScript and builds the webview
7. **Package extension** - Creates the .vsix file
8. **Publish to marketplace** - Publishes the extension (only on tag pushes)
9. **Upload artifact** - Saves the .vsix file as a GitHub artifact

## Version Management

- The workflow uses the version from `package.json`
- For manual publishing, you can specify a custom version
- Tag-based publishing uses the tag version

## Troubleshooting

### Common Issues

1. **PAT not found**: Ensure `VSCE_PAT` secret is set correctly
2. **Build failures**: Check that all dependencies are properly installed
3. **Publish failures**: Verify your PAT has publish permissions for the temporal-technologies publisher

### Debugging

- Check the workflow logs in the Actions tab
- Download the extension artifact to test locally
- Verify the .vsix file is created correctly

## Security

- The PAT is stored as a GitHub secret and never exposed in logs
- Only authorized users can trigger manual workflows
- Automatic publishing only occurs on version tags 
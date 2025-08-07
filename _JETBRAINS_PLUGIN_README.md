# JetBrains Plugin Publishing Workflow

This workflow automatically publishes the Temporal Workflow Debugger plugin to the JetBrains marketplace.

## Setup

### 1. Create a JetBrains Marketplace Account

1. Go to [JetBrains Marketplace](https://plugins.jetbrains.com/)
2. Sign in with your JetBrains account
3. Create a publisher account if you don't have one
4. Create a new plugin listing for "Temporal Workflow Debugger"

### 2. Generate Plugin Signing Certificate

1. Generate a certificate for plugin signing:
   ```bash
   keytool -genkey -alias temporal-plugin -keyalg RSA -keysize 2048 -validity 10000 -keystore temporal-plugin.jks
   ```

2. Export the certificate chain and private key:
   ```bash
   # Export certificate chain
   keytool -export -alias temporal-plugin -file temporal-plugin.crt -keystore temporal-plugin.jks
   
   # Export private key (you'll need to convert to PEM format)
   keytool -importkeystore -srckeystore temporal-plugin.jks -destkeystore temporal-plugin.p12 -deststoretype PKCS12
   openssl pkcs12 -in temporal-plugin.p12 -out temporal-plugin.pem -nodes
   ```

### 3. Get JetBrains Publish Token

1. Go to your JetBrains Marketplace publisher account
2. Navigate to "API Keys" section
3. Generate a new API key for publishing

### 4. Add GitHub Secrets

1. Go to your GitHub repository settings
2. Navigate to "Secrets and variables" → "Actions"
3. Add the following repository secrets:

   **Required for publishing:**
   - **Name**: `JETBRAINS_PUBLISH_TOKEN`
   - **Value**: Your JetBrains marketplace API key

   **Required for plugin signing:**
   - **Name**: `CERTIFICATE_CHAIN`
   - **Value**: Your certificate chain (PEM format)
   - **Name**: `PRIVATE_KEY`
   - **Value**: Your private key (PEM format)
   - **Name**: `PRIVATE_KEY_PASSWORD`
   - **Value**: Your private key password

## Usage

### Automatic Publishing (Recommended)

The workflow automatically publishes when you:

1. **Push a tag** starting with 'v' (e.g., `v1.0.0`, `v2.1.3`)
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Push to main branch** (will build and package but not publish)

### Manual Publishing

1. Go to the "Actions" tab in your GitHub repository
2. Select "Publish JetBrains Plugin" workflow
3. Click "Run workflow"
4. Enter the version number (e.g., `1.0.0`)
5. Click "Run workflow"

## Workflow Steps

1. **Checkout code** - Gets the latest code
2. **Setup JDK 17** - Installs Java 17 with Gradle caching
3. **Setup Gradle** - Configures Gradle build environment
4. **Build plugin** - Compiles the Java code
5. **Run tests** - Executes the test suite
6. **Create plugin package** - Builds the plugin with signing
7. **Upload artifact** - Saves the plugin as a GitHub artifact
8. **Publish to marketplace** - Publishes the plugin (only on tag pushes)
9. **Manual publish** - Publishes with custom version (manual trigger)

## Version Management

- The workflow uses the version from `build.gradle.kts`
- For manual publishing, you can specify a custom version
- Tag-based publishing uses the tag version
- The plugin version is automatically updated in the build file for manual releases

## Plugin Configuration

The plugin is configured in `jetbrains-plugin/build.gradle.kts`:

- **Plugin ID**: `com.temporal.wf-debugger`
- **Plugin Name**: Temporal Workflow Debugger
- **Vendor**: Temporal Technologies
- **IDE Compatibility**: GoLand 2024.1+ (241-243.*)

## Troubleshooting

### Common Issues

1. **Publish token not found**: Ensure `JETBRAINS_PUBLISH_TOKEN` secret is set correctly
2. **Certificate issues**: Verify certificate chain and private key are in correct PEM format
3. **Build failures**: Check that all dependencies are properly configured
4. **Version conflicts**: Ensure version numbers follow semantic versioning

### Debugging

- Check the workflow logs in the Actions tab
- Download the plugin artifact to test locally
- Verify the plugin builds correctly with `./gradlew buildPlugin`
- Test the plugin in a local IDE before publishing

## Security

- All sensitive data (tokens, certificates) are stored as GitHub secrets
- Only authorized users can trigger manual workflows
- Automatic publishing only occurs on version tags
- Plugin signing ensures authenticity and integrity

## Plugin Features

The Temporal Workflow Debugger plugin provides:

- Workflow history upload and visualization
- Debugging configuration management
- Integration with tdlv debugger
- Breakpoint management on history events
- Tool window for workflow debugging
- Run configuration for debugging workflows

## Compatibility

- **Supported IDEs**: GoLand, IntelliJ IDEA, and other JetBrains IDEs
- **Minimum IDE Version**: 2024.1 (241)
- **Maximum IDE Version**: 2024.3+ (243.*)
- **Java Version**: 17+
- **Gradle Version**: 8.0+ 
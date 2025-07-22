plugins {
    id("java")
    id("org.jetbrains.intellij") version "1.17.2"
}

group = "com.temporal"
version = "1.0-SNAPSHOT"

repositories {
    mavenCentral()
}

dependencies {
    implementation("com.fasterxml.jackson.core:jackson-core:2.15.2")
    implementation("com.fasterxml.jackson.core:jackson-databind:2.15.2")
    implementation("com.fasterxml.jackson.core:jackson-annotations:2.15.2")
    testImplementation("junit:junit:4.13.2")

    // Removed Go plugin compile-time dependencies since we use reflection
    // to access Go plugin classes at runtime only when they're available
}

// Configure Gradle IntelliJ Plugin
// Read more: https://plugins.jetbrains.com/docs/intellij/tools-gradle-intellij-plugin.html
intellij {
    version.set("2024.3.1")  // Updated to newer version for better compatibility
    type.set("GO") // GoLand IDE

    // No longer need Go plugin at build time since we use reflection
    plugins.set(listOf())  // Remove go plugin dependency
}

tasks {
    // Set the JVM compatibility versions
    withType<JavaCompile> {
        sourceCompatibility = "17"
        targetCompatibility = "17"
        options.compilerArgs.addAll(listOf("--add-modules", "jdk.httpserver"))
    }

    patchPluginXml {
        sinceBuild.set("241")
        untilBuild.set("243.*")  // Support newer IDE versions including 2024.3 and beyond
    }

    signPlugin {
        certificateChain.set(System.getenv("CERTIFICATE_CHAIN"))
        privateKey.set(System.getenv("PRIVATE_KEY"))
        password.set(System.getenv("PRIVATE_KEY_PASSWORD"))
    }

    publishPlugin {
        token.set(System.getenv("PUBLISH_TOKEN"))
    }
} 

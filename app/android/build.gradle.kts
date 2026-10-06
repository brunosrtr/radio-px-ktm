allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

// ndk-build não escapa espaços no caminho de saída (ex.: Área de trabalho).
// Apenas os intermediários do Opus usam o cache Gradle, isolados por projeto.
val opusBuildDir = gradle.gradleUserHomeDir.resolve(
    "radio-px-native/" + Integer.toHexString(rootProject.projectDir.absolutePath.hashCode())
)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    if (project.name == "opus_flutter_new_android") {
        project.layout.buildDirectory.set(opusBuildDir)
        // O plugin declara SDK 33; o embedding Flutter atual exige SDK >= 34.
        project.afterEvaluate {
            extensions.configure<com.android.build.gradle.LibraryExtension> {
                compileSdk = 36
            }
        }
    } else {
        project.layout.buildDirectory.value(newSubprojectBuildDir)
    }
}
subprojects {
    project.evaluationDependsOn(":app")
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
    delete(opusBuildDir)
}

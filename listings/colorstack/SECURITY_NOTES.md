# Security Notes

Release signing material must not be committed to the repository.

## Local setup for release signing

1. Keep the release keystore file outside Git, or place it locally under `android/` only on the build machine.
2. Copy `android/key.properties.example` to `android/key.properties`.
3. Fill in the local signing values on the build machine or inject them through CI secrets.
4. Do not commit `android/key.properties`, `*.jks`, or `*.keystore` files.

The Gradle release build falls back to the debug signing config when `android/key.properties` is absent. This allows development checks to continue without exposing production signing secrets.

If the old keystore and password were already shared in source control, rotate the app signing key according to the distribution platform's process.

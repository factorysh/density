# Compose

The package `compose` handles `docker-compose` tool, with validation, patching and running.

`ComposeValidator` groups a collection of `VolumeValidator` and `ServiceValidator` and validate a `docker-compose.yml` file.

`Recomposator`groups a collection of `VolumePatcher` and `ServicePatcher`and create a patched `docker-compose.yml` file.

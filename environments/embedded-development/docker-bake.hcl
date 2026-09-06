group "default" {
  targets = ["mcu-dev"]
}

group "all" {
  targets = ["base", "mcu-dev"]
}

group "verify-arm64" {
  targets = ["base-arm64", "mcu-dev-arm64"]
}

group "verify-amd64" {
  targets = ["base-amd64", "mcu-dev-amd64"]
}

group "multiarch" {
  targets = ["base-multiarch", "mcu-dev-multiarch"]
}

group "publish" {
  targets = ["mcu-dev-multiarch"]
}

target "common" {
  context    = "."
  dockerfile = "environments/embedded-development/Dockerfile"
}

target "base" {
  inherits = ["common"]
  target   = "base"
  tags     = ["mcu-dev/base:local"]
}

target "mcu-dev" {
  inherits = ["common"]
  target   = "mcu-dev"
  tags     = ["mcu-dev/toolchain:local"]
}

target "base-arm64" {
  inherits  = ["base"]
  platforms = ["linux/arm64"]
}

target "mcu-dev-arm64" {
  inherits  = ["mcu-dev"]
  platforms = ["linux/arm64"]
}

target "base-amd64" {
  inherits  = ["base"]
  platforms = ["linux/amd64"]
}

target "mcu-dev-amd64" {
  inherits  = ["mcu-dev"]
  platforms = ["linux/amd64"]
}

target "base-multiarch" {
  inherits  = ["base"]
  platforms = ["linux/amd64", "linux/arm64"]
}

target "mcu-dev-multiarch" {
  inherits  = ["mcu-dev"]
  platforms = ["linux/amd64", "linux/arm64"]
}

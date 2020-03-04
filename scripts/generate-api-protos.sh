#!/usr/bin/env bash
set -euo pipefail

# https://github.com/protocolbuffers/protobuf/releases
readonly PROTOC_RELEASE=3.11.4
PROTO_ZIP_RELEASE_MD5_LINUX=7c0babfc7d2ae4eff6ce3e47c2de90c2
PROTO_ZIP_RELEASE_MD5_OSX=58c8716eabdbc1259d14880ace6e719a

# Infer paths.
readonly REPO_ROOT="$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")"
readonly API_ROOT="${REPO_ROOT}/api"

# Isolate `go install` commands to the repository. We'll also use this path for other binaries.
export GOBIN="${REPO_ROOT}/bin"

# Get the directory that the go module is stored in.
modpath() {
  set -e

  # Ensure we have the correct version of the module installed.
  go mod download "${1}"

  go list -f "{{ .Dir }}" -m "${1}"
}

# Excecute protoc with all the trimmings.
protoc_command() {
  echo "${1}"
  "${PROTOC_BIN}" "${IPATHS[@]}" \
    --go_out=paths=source_relative:./pkg \
    --plugin=protoc-gen-validate="${GOBIN}/protoc-gen-validate" \
    --validate_out="lang=go:pkg/" \
    --plugin=protoc-gen-go="${GOBIN}/protoc-gen-go" \
    "${1}"
}

protoc_lint() {
  "${PROTOC_BIN}" "${IPATHS[@]}" \
    --buf-check-lint_out=. \
    "--buf-check-lint_opt={\"input_config\": ${BUF_LINT_CONFIG}}" \
    --plugin=protoc-gen-buf-check-lint="${GOBIN}/protoc-gen-buf-check-lint" \
    "${1}" 2>&1
}

main() {
  # Install required plugins for generation. These deps are pinned in go.mod.
  go install \
    github.com/envoyproxy/protoc-gen-validate \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
    github.com/golang/protobuf/protoc-gen-go \
    github.com/go-swagger/go-swagger/cmd/swagger \
    github.com/bufbuild/buf/cmd/protoc-gen-buf-check-lint

  # Include .proto deps from go modules.
  IPATHS=(
    "${PROTOC_INCLUDE}"
    "${API_ROOT}"
    "$(modpath github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis"
    "$(modpath github.com/envoyproxy/protoc-gen-validate)"
  )

  for i in "${!IPATHS[@]}"; do
    include_folder="${IPATHS[$i]}"
    if [[ ! -d "${include_folder}" ]]; then
      echo "error: bad proto_path include '${include_folder}'"
      exit 1
    fi
    IPATHS[$i]="-I${include_folder}"
  done

  # Collect a list of protos.
  # The while loop syntax below is from https://github.com/koalaman/shellcheck/wiki/SC2044 due to lack of bash 4 on OSX.
  PROTOS=()
  while IFS= read -r -d '' proto; do
    PROTOS+=("${proto}")
  done <  <(find "${API_ROOT}" -name '*.proto' -print0)

  # Lint and exit if the lint argument is provided.
  if [[ "${1-}" == "lint"* ]]; then
    BUF_LINT_CONFIG=$(cat "${REPO_ROOT}/buf.json")

    if [[ ! -f "${CLANG_FORMAT_BIN}" ]]; then
      echo "error: clang-format not available, please run \`yarn install\`."
      exit 1
    fi

    set +e  # Count the failures ourselves instead of early exit so user can see all lint failures.
    lint_ok=true

    # buf lint. errors are not auto-fixable, so we output the issues with them regardless.
    for proto in "${PROTOS[@]}"; do
      if ! output=$(protoc_lint "${proto}"); then
        echo "--- ${proto}"
        echo "${output}" | sed 's/--buf-check-lint_out: //' | cut -d":" -f2-
        lint_ok=false
      fi
    done

    # clang-format for whitespace.
    if [[ "${1-}" == "lint:fix" ]]; then
      # Fix whitespace in protos.
      for proto in "${PROTOS[@]}"; do
        "${CLANG_FORMAT_BIN}" -i "${proto}"
      done
    else
      # Check for diffs in formatted and existing proto.
      for proto in "${PROTOS[@]}"; do
        if ! output=$("${CLANG_FORMAT_BIN}" "${proto}" | diff -u "${proto}" -); then
          echo "${output}"
          lint_ok=false
        fi
      done
    fi

    ${lint_ok} && exit 0 || exit 1
  fi

  echo "Compiling protos using $("${PROTOC_BIN}" --version) for protoc..."
  for proto in "${PROTOS[@]}"; do
    protoc_command "${proto}"
  done
}

# Usage md5check <desired_md5> <filename>.
md5check() {
  desired_md5="${1}"
    filename="${2}"
  file_md5=$(md5sum "${filename}" | cut -d' ' -f1)

  if [[ "${file_md5}" != "${desired_md5}" ]]; then
    echo "error: MD5 sum of '${filename}' was ${file_md5} instead of ${desired_md5}."
    exit 1
  fi
}

### Set up dependencies.
mkdir -p "${GOBIN}"

# Install clang-format.
CLANG_FORMAT_BIN="${GOBIN}/clang-format"
if [[ ! -f "${CLANG_FORMAT_BIN}" ]]; then
  echo "info: Downloading clang-format to build environment"

  case "${OSTYPE}" in
    "darwin"*) clang_format_os="darwin"; clang_format_md5="c3ebe742599dcc38b9dc6544cacd69bb" ;;
    "linux"*) clang_format_os="linux"; clang_format_md5="fee8c52e196e28ae5928d6ff8757f58c" ;;
    *) echo "error: Unsupported OS '${OSTYPE}' for clang-format install, please install manually" && exit 1 ;;
  esac

  curl -sSL -o "/tmp/clang-format" \
    "https://github.com/angular/clang-format/raw/v1.4.0/bin/${clang_format_os}_x64/clang-format"
  md5check "${clang_format_md5}" "/tmp/clang-format"
  chmod a+x "/tmp/clang-format"
  mv "/tmp/clang-format" "${CLANG_FORMAT_BIN}"

fi

# Install or update protobuf.
if [[ -f /etc/alpine-release ]]; then
  # Alpine must use it's built-in version of protoc since the published version uses glibc and does not support musl.
  # Once we dump alpine in Docker we can delete.
  PROTOC_BIN=$(command -v protoc)
  PROTOC_INCLUDE=/usr/local/include
else
  PROTOC_BIN="${GOBIN}/protoc-v${PROTOC_RELEASE}"
  PROTOC_INCLUDE="${GOBIN}/protoc-v${PROTOC_RELEASE}-include"
fi

if [[ ! -f "${PROTOC_BIN}" || ! -d "${PROTOC_INCLUDE}" ]]; then
  echo "info: Downloading protoc-v${PROTOC_RELEASE} to build environment"

  # cleanup old versions
  find "${GOBIN}" -regex '.*/protoc-v[0-9.]+$' -exec rm {} \;

  case "${OSTYPE}" in
    "darwin"*) proto_os="osx"; proto_md5="${PROTO_ZIP_RELEASE_MD5_OSX}" ;;
    "linux"*) proto_os="linux"; proto_md5="${PROTO_ZIP_RELEASE_MD5_LINUX}" ;;
    *) echo "error: Unsupported OS '${OSTYPE}' for protoc install, please install manually" && exit 1 ;;
  esac
  proto_arch=x86_64
  proto_zip_out="/tmp/protoc-${PROTOC_RELEASE}.zip"
  curl -sSL -o "${proto_zip_out}" \
    "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_RELEASE}/protoc-${PROTOC_RELEASE}-${proto_os}-${proto_arch}.zip"
  md5check "${proto_md5}" "${proto_zip_out}"

  proto_dir_out="/tmp/proto-${PROTOC_RELEASE}"
  mkdir -p "${proto_dir_out}"
  unzip -q -o "${proto_zip_out}" -d "${proto_dir_out}"

  mv "${proto_dir_out}"/bin/protoc "${PROTOC_BIN}"
  mv "${proto_dir_out}"/include "${PROTOC_INCLUDE}"
fi

main "$@"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)}"

mkdir -p "${DIST_DIR}"
rm -rf "${DIST_DIR:?}/"*

build_target() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  local out_dir="${DIST_DIR}/codex-proxy_${VERSION}_${goos}_${goarch}"
  local bin_name="codex-proxy"

  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
    bin_name="codex-proxy.exe"
  fi

  mkdir -p "${out_dir}"
  echo "==> Building ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o "${out_dir}/${bin_name}" ./cmd/codex-proxy

  cp "${ROOT_DIR}/README.md" "${out_dir}/README.md"
  cp "${ROOT_DIR}/LICENSE" "${out_dir}/LICENSE"
  cp "${ROOT_DIR}/CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md" "${out_dir}/CODEX_PROXY_FIX_IMPLEMENTATION_BLUEPRINT.md"
  cp "${ROOT_DIR}/RELEASE_READINESS_PLAN.md" "${out_dir}/RELEASE_READINESS_PLAN.md"

  if [[ "${goos}" == "windows" ]]; then
    (cd "${DIST_DIR}" && zip -rq "codex-proxy_${VERSION}_${goos}_${goarch}.zip" "$(basename "${out_dir}")")
  else
    tar -C "${DIST_DIR}" -czf "${DIST_DIR}/codex-proxy_${VERSION}_${goos}_${goarch}.tar.gz" "$(basename "${out_dir}")"
  fi

  rm -rf "${out_dir}"
}

build_target darwin arm64
build_target darwin amd64
build_target linux amd64
build_target linux arm64
build_target windows amd64
build_target windows arm64

(cd "${DIST_DIR}" && find . -maxdepth 1 -type f ! -name 'checksums.txt' -print0 | xargs -0 shasum -a 256 > checksums.txt)
echo "Artifacts written to ${DIST_DIR}"

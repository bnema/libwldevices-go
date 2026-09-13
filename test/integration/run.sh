#!/usr/bin/env bash
#
# Headless wlroots compositor integration harness for libwldevices-go.
#
# What it does:
#   1. builds the digest-pinned fixture image (test/integration/Containerfile),
#   2. builds the standalone consumer module (test/consumer) on the host,
#   3. starts sway headless inside a detached container with the consumer
#      binary bind-mounted read-only,
#   4. waits, with a bounded timeout, for the compositor's Wayland socket,
#   5. runs the consumer inside the container against that socket.
#
# The consumer's exit status is propagated, the container is always removed and
# the compositor log is printed whenever the run fails.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CONSUMER_DIR="${REPO_ROOT}/test/consumer"

IMAGE_TAG="wlvision-libwldevices-fixture:local"
RUNTIME_DIR="/run/user/1000"
SOCKET_TIMEOUT_SECS="${SOCKET_TIMEOUT_SECS:-30}"

BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/libwldevices-fixture.XXXXXX")"
CID=""

cleanup() {
	local status=$?
	set +e
	if [[ -n "${CID}" ]]; then
		if [[ ${status} -ne 0 ]]; then
			echo "--- compositor container log (run failed, exit ${status}) ---" >&2
			docker logs "${CID}" 2>&1 || true
			echo "--- end compositor container log ---" >&2
		fi
		docker rm -f "${CID}" >/dev/null 2>&1 || true
	fi
	rm -rf "${BUILD_DIR}"
	return "${status}"
}
trap cleanup EXIT

if ! command -v docker >/dev/null 2>&1; then
	echo "ERROR: docker CLI not found in PATH" >&2
	exit 1
fi

echo "==> building fixture image ${IMAGE_TAG}"
docker build -q -t "${IMAGE_TAG}" -f "${SCRIPT_DIR}/Containerfile" "${SCRIPT_DIR}" >/dev/null

echo "==> building consumer binary (standalone module)"
(
	cd "${CONSUMER_DIR}"
	# The committed module file stays release-resolved: the local replacements
	# live in a throwaway alternate module file that is removed again below.
	cp go.mod go.local.mod
	cp go.sum go.local.sum
	go mod edit -modfile=go.local.mod \
		-replace "github.com/bnema/libwldevices-go=../.." \
		-replace "github.com/bnema/wlturbo=../../../wlturbo"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -modfile=go.local.mod -o "${BUILD_DIR}/consumer" .
	rm -f go.local.mod go.local.sum
)

echo "==> starting headless sway compositor"
CID="$(docker run -d -v "${BUILD_DIR}/consumer:/consumer:ro" "${IMAGE_TAG}" sh -c '
set -e
export XDG_RUNTIME_DIR=/run/user/1000
export WLR_BACKENDS=headless
export WLR_RENDERER=pixman
export WLR_LIBINPUT_NO_DEVICES=1
: > /tmp/sway-empty.conf
exec sway -c /tmp/sway-empty.conf
')"
echo "    container ${CID}"

echo "==> waiting up to ${SOCKET_TIMEOUT_SECS}s for the compositor socket"
WAYLAND_DISPLAY=""
deadline=$((SECONDS + SOCKET_TIMEOUT_SECS))
while ((SECONDS < deadline)); do
	if ! docker inspect -f '{{.State.Running}}' "${CID}" 2>/dev/null | grep -q true; then
		echo "ERROR: compositor container exited before publishing a Wayland socket" >&2
		exit 1
	fi
	sock="$(docker exec "${CID}" sh -c 'for s in /run/user/1000/wayland-?; do [ -S "$s" ] && basename "$s" && break; done' 2>/dev/null || true)"
	if [[ -n "${sock}" ]]; then
		WAYLAND_DISPLAY="${sock}"
		break
	fi
	sleep 0.2
done

if [[ -z "${WAYLAND_DISPLAY}" ]]; then
	echo "ERROR: timed out after ${SOCKET_TIMEOUT_SECS}s waiting for the compositor socket" >&2
	docker exec "${CID}" sh -c 'ls -la /run/user/1000' >&2 2>&1 || true
	exit 1
fi
echo "    compositor socket ready: WAYLAND_DISPLAY=${WAYLAND_DISPLAY}"

echo "==> running consumer inside the container"
set +e
docker exec \
	-e "XDG_RUNTIME_DIR=${RUNTIME_DIR}" \
	-e "WAYLAND_DISPLAY=${WAYLAND_DISPLAY}" \
	"${CID}" /consumer
consumer_status=$?
set -e

if [[ ${consumer_status} -ne 0 ]]; then
	echo "ERROR: consumer exited with status ${consumer_status}" >&2
	exit "${consumer_status}"
fi

echo "==> integration smoke passed"

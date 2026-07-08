#!/usr/bin/env python3

import argparse
import os
import shlex
import subprocess
import sys
from pathlib import Path


SECRET_ARG_PREFIXES = (
    "LOGGING_SLACK_TOKEN=",
)


def env_value(name, default=None):
    value = os.environ.get(name)
    if value is None or value == "":
        return default
    return value


def env_bool(name, default):
    value = env_value(name)
    if value is None:
        return default
    return value.lower() in ("1", "true", "yes", "on")


def redact_arg(arg):
    for prefix in SECRET_ARG_PREFIXES:
        if arg.startswith(prefix):
            return prefix + "***"
    return arg


def print_command(command):
    rendered = " ".join(shlex.quote(redact_arg(part)) for part in command)
    print(f"+ {rendered}", flush=True)


def append_log(log_file, text):
    if not log_file:
        return

    log_file.parent.mkdir(parents=True, exist_ok=True)
    with log_file.open("a", encoding="utf-8", errors="replace") as stream:
        stream.write(text)
        if not text.endswith("\n"):
            stream.write("\n")


def reset_log(log_file):
    if not log_file:
        return

    log_file.parent.mkdir(parents=True, exist_ok=True)
    log_file.write_text("", encoding="utf-8", errors="replace")


def run(command, cwd):
    print_command(command)
    log_file = Path(os.environ["CI_BUILD_LOG"]) if env_value("CI_BUILD_LOG") else None
    rendered = " ".join(shlex.quote(redact_arg(part)) for part in command)
    append_log(log_file, f"+ {rendered}")

    with subprocess.Popen(
        command,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        encoding="utf-8",
        errors="replace",
        bufsize=1,
    ) as process:
        assert process.stdout is not None
        for line in process.stdout:
            print(line, end="", flush=True)
            append_log(log_file, line)

    if process.returncode:
        raise subprocess.CalledProcessError(process.returncode, command)


def docker_context_path(path, context_dir):
    source_path = Path(path)
    if not source_path.is_absolute():
        source_path = context_dir / source_path

    source_path = source_path.resolve()
    context_dir = context_dir.resolve()

    try:
        relative_path = source_path.relative_to(context_dir)
    except ValueError:
        raise SystemExit(
            f"Secret path must be inside the Docker build context: {source_path}"
        )

    return relative_path.as_posix()


def required_prod_secret_paths(secrets_dir, context_dir):
    required_files = (
        "libuserkey.a",
        "userkey.h",
        "user_key.go",
        "cert-edrlab.pem",
        "privkey-edrlab.pem",
    )

    paths = {}
    for filename in required_files:
        source = Path(secrets_dir) / filename
        resolved = source if source.is_absolute() else context_dir / source
        if not resolved.is_file():
            raise SystemExit(f"Missing prod secret: {source}")
        paths[filename] = docker_context_path(source, context_dir)

    return paths


def build_command(args):
    build_args = [
        "docker",
        "build",
        "--progress=plain",
        "-f",
        args.dockerfile,
        "--build-arg",
        f"BUILD_PROD={'true' if args.mode == 'prod' else 'false'}",
    ]

    if args.platform.lower() not in ("", "local", "none"):
        build_args.extend(["--platform", args.platform])

    if args.slack_token:
        build_args.extend(["--build-arg", f"LOGGING_SLACK_TOKEN={args.slack_token}"])

    if args.slack_token and not args.slack_channel:
        args.slack_channel = "C07KFNG59PF"

    if args.slack_channel:
        build_args.extend(["--build-arg", f"LOGGING_SLACK_CHANNEL={args.slack_channel}"])

    if args.mode == "prod":
        secret_paths = required_prod_secret_paths(args.secrets_dir, args.context_dir)
        build_args.extend(
            [
                "--build-arg",
                f"LIBUSERKEY_PATH={secret_paths['libuserkey.a']}",
                "--build-arg",
                f"USERKEYH_PATH={secret_paths['userkey.h']}",
                "--build-arg",
                f"USERKEYGO_PATH={secret_paths['user_key.go']}",
                "--build-arg",
                f"PRIVATE_KEY_PATH={secret_paths['privkey-edrlab.pem']}",
                "--build-arg",
                f"CERTIFICATE_PATH={secret_paths['cert-edrlab.pem']}",
                "--build-arg",
                f"PROFILE={args.profile}",
            ]
        )

    build_args.extend(["-t", args.image_ref, str(args.context_dir)])
    return build_args


def test_command(args):
    command = ["docker", "run", "--rm"]

    if args.platform.lower() not in ("", "local", "none"):
        command.extend(["--platform", args.platform])

    command.extend(
        [
            "--entrypoint",
            "sh",
            args.image_ref,
            "-c",
            (
                "test -x /usr/bin/lcpserver "
                "&& test -x /usr/bin/lsdserver "
                "&& test -x /usr/bin/frontend "
                "&& for file in /etc/inittab /init.sh /etc/nginx/templates/default.conf.template; do "
                "tr -d '\\r' < \"$file\" | cmp -s \"$file\" - "
                "|| { echo \"$file contains CR characters\"; exit 1; }; "
                "done "
                "&& grep -q 'absolute_redirect off;' /etc/nginx/templates/default.conf.template "
                "&& grep -q 'port_in_redirect off;' /etc/nginx/templates/default.conf.template "
                "&& grep -q 'return 302 /frontend/;' /etc/nginx/templates/default.conf.template "
                '&& echo "Tests passed"'
            ),
        ]
    )
    return command


def parse_args():
    parser = argparse.ArgumentParser(
        description="Build and smoke-test the Readium LCP server Docker image."
    )
    parser.add_argument("mode", nargs="?", choices=("test", "prod"), default="test")
    parser.add_argument(
        "secrets_dir",
        nargs="?",
        default=env_value("SECRETS_DIR", "./_"),
        help="Directory containing prod secrets. Must be inside the Docker context.",
    )
    parser.add_argument(
        "--context",
        dest="context_dir",
        default=env_value("BUILD_CONTEXT", "."),
        type=Path,
    )
    parser.add_argument(
        "--dockerfile",
        default=env_value("DOCKERFILE", "docker/Dockerfile"),
    )
    parser.add_argument(
        "--image-name",
        default=None,
    )
    parser.add_argument(
        "--tag",
        default=env_value("TAG", "latest"),
    )
    parser.add_argument(
        "--image-ref",
        default=env_value("IMAGE_REF"),
    )
    parser.add_argument(
        "--platform",
        default=env_value("PLATFORM", "linux/amd64"),
        help="Docker platform. Use 'local' to let Docker choose the host platform.",
    )
    parser.add_argument(
        "--profile",
        default=env_value("PROFILE"),
    )
    parser.add_argument(
        "--skip-test",
        action="store_true",
        default=not env_bool("RUN_IMAGE_TEST", True),
    )

    args = parser.parse_args()

    default_image_name = f"readium-lcp-server-{args.mode}"
    image_name = args.image_name or env_value("IMAGE_NAME", default_image_name)
    args.image_ref = args.image_ref or f"{image_name}:{args.tag}"
    args.profile = args.profile or ("2.2" if args.mode == "prod" else "basic")
    args.context_dir = args.context_dir.resolve()
    args.slack_token = env_value("LOGGING_SLACK_TOKEN")
    args.slack_channel = env_value("LOGGING_SLACK_CHANNEL")

    return args


def main():
    args = parse_args()

    log_file = Path(os.environ["CI_BUILD_LOG"]) if env_value("CI_BUILD_LOG") else None
    reset_log(log_file)
    print(
        f"Building {args.image_ref} from {args.dockerfile} in {args.mode} mode",
        flush=True,
    )
    append_log(
        log_file,
        f"Building {args.image_ref} from {args.dockerfile} in {args.mode} mode",
    )
    run(build_command(args), cwd=args.context_dir)

    if not args.skip_test:
        print(f"Testing {args.image_ref}", flush=True)
        append_log(log_file, f"Testing {args.image_ref}")
        run(test_command(args), cwd=args.context_dir)

    print(f"Image ready locally: {args.image_ref}", flush=True)
    append_log(log_file, f"Image ready locally: {args.image_ref}")


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as error:
        sys.exit(error.returncode)

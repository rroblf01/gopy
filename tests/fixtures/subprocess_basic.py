import subprocess


def main() -> None:
    # `echo` exists on POSIX systems used by the test harness. The gopy
    # shim ignores capture_output / text kwargs — output is always
    # captured as a string.
    result = subprocess.run(["echo", "hello gopy"], capture_output=True, text=True)
    print(result.returncode)
    print(result.stdout.strip())


if __name__ == "__main__":
    main()

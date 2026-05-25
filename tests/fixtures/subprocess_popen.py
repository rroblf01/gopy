import subprocess


def main() -> None:
    p = subprocess.Popen(
        ["sh", "-c", "echo hello"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    parts = p.communicate()
    out = str(parts[0]).strip()
    print(out)
    print(p.returncode)


if __name__ == "__main__":
    main()

import subprocess


def main() -> None:
    # Pipe input through cat; verify stdin made it.
    r = subprocess.run(["cat"], input="from-stdin\n", capture_output=True, text=True)
    print(r.stdout.strip())
    print(r.returncode)


if __name__ == "__main__":
    main()

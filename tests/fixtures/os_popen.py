import os


def main() -> None:
    fh = os.popen("echo hello")
    body: str = fh.read()
    print(body.strip())


if __name__ == "__main__":
    main()

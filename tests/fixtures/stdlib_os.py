import os


def main() -> None:
    # PATH is universally set in both CPython and the Go binary environments;
    # we just check that the lookup returns something non-empty.
    p: str = os.getenv("PATH")
    print(len(p) > 0)


if __name__ == "__main__":
    main()

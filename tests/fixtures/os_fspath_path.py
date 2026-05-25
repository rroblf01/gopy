import os
from pathlib import Path


def main() -> None:
    p = Path("/tmp/foo.txt")
    print(os.fspath(p))
    print(os.fspath("/tmp/bar.txt"))


if __name__ == "__main__":
    main()

from pathlib import Path


def main() -> None:
    p = Path("archive.tar.gz")
    for s in p.suffixes:
        print(s)


if __name__ == "__main__":
    main()

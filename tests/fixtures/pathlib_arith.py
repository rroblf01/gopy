from pathlib import Path


def main() -> None:
    base = Path("/tmp/gopy")
    sub = base / "data"
    leaf = sub / "file.txt"
    print(str(leaf))
    print(leaf.name)
    print(str(leaf.parent))


if __name__ == "__main__":
    main()

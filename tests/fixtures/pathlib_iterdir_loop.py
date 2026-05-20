import os
from pathlib import Path


def main() -> None:
    base = Path("/tmp/gopy_iterdir_test")
    os.makedirs("/tmp/gopy_iterdir_test", exist_ok=True)
    a = base / "a.txt"
    b = base / "b.txt"
    a.write_text("x")
    b.write_text("y")
    names: list[str] = []
    for child in base.iterdir():
        names.append(child.name)
    names.sort()
    for n in names:
        print(n)
    a.unlink()
    b.unlink()


if __name__ == "__main__":
    main()
